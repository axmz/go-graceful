package graceful

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var logger = slog.Default().With("component", "graceful_shutdown")

// SetLogger allows setting a custom logger or disable logging altogether by passing nil.
func SetLogger(l *slog.Logger) {
	if l == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	} else {
		logger = l
	}
}

// Operation defines a function type for cleanup operations that can be executed
// during the shutdown process. It takes a context and returns an error if any.
type Operation = func(ctx context.Context) error

// Shutdown gracefully shuts down the application by executing cleanup operations
// defined in the ops map. It listens for specified OS signals and waits for
// the context to be done or the timeout to expire.
func Shutdown(
	ctx context.Context,
	timeout time.Duration,
	ops map[string]Operation,
	signals ...os.Signal,
) (<-chan struct{}, <-chan error) {
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
	}

	wait := make(chan struct{})
	errs := make(chan error, len(ops))

	go func() {
		s := make(chan os.Signal, 1)
		signal.Notify(s, signals...)

		select {
		case sig := <-s:
			logger.Info("Shutting down due to signal", "signal", sig)
		case <-ctx.Done():
			logger.Info("Shutting down due to context cancellation", "reason", ctx.Err())
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		var wg sync.WaitGroup
		for key, op := range ops {
			wg.Add(1)
			go func(key string, op Operation) {
				defer wg.Done()
				if err := op(shutdownCtx); err != nil {
					errs <- err
				}
			}(key, op)
		}
		wg.Wait()

		close(wait)
	}()

	return wait, errs
}
