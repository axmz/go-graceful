package graceful

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func init() {
	// Disable logging during tests
	SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestShutdown_ContextCancellation(t *testing.T) {
	var called atomic.Bool

	ctx, cancel := context.WithCancel(context.Background())

	ops := map[string]Operation{
		"op1": func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
	}

	waitCh, errCh := Shutdown(ctx, time.Second, ops)

	cancel()

	select {
	case <-waitCh:
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}

	if !called.Load() {
		t.Error("Expected operation to be called on context cancellation")
	}

	select {
	case err := <-errCh:
		t.Errorf("Unexpected error: %v", err)
	default:
		// no errors expected
	}
}

func TestShutdown_SignalTrigger(t *testing.T) {
	var called atomic.Bool

	ops := map[string]Operation{
		"op1": func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
	}

	ctx := context.Background()
	waitCh, errCh := Shutdown(ctx, time.Second, ops, syscall.SIGUSR1)

	time.Sleep(100 * time.Millisecond)
	// Send the signal to self
	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)

	select {
	case <-waitCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete in time after signal")
	}

	if !called.Load() {
		t.Error("Expected operation to be called on signal")
	}

	select {
	case err := <-errCh:
		t.Errorf("Unexpected error: %v", err)
	default:
	}
}

func TestShutdown_OperationError(t *testing.T) {
	expectedErr := errors.New("op failed")

	ops := map[string]Operation{
		"fail": func(ctx context.Context) error {
			return expectedErr
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	waitCh, errCh := Shutdown(ctx, time.Second, ops)
	cancel()

	select {
	case <-waitCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete")
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}
	default:
		t.Error("Expected an error but got none")
	}
}

func TestShutdown_Timeout(t *testing.T) {
	start := time.Now()

	ops := map[string]Operation{
		"sleep": func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second): // will exceed timeout
				return nil
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	waitCh, errCh := Shutdown(ctx, 100*time.Millisecond, ops)
	cancel()

	select {
	case <-waitCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete")
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context deadline exceeded, got %v", err)
		}
	default:
		t.Error("Expected an error due to timeout")
	}

	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Error("Test took too long, timeout likely not respected")
	}
}
