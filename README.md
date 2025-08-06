# go-graceful
go-graceful pkg listens to signals or context cancelation to initiate shutdown.

```go 
ctx, cancel := context.WithCancel(context.Background())

// a failed service calls 
cancel()
// or a signal is sent
// thus triggering graceful shutdown

wait, errs := Shutdown(ctx, timeout, map[string]graceful.Operation{
	"kafka":       ShutdownKafka,
	"database":    ShutdownDB,
	"http-server": ShutdownHTTP,
})

for {
	select {
	case <-wait:
		// Shutdown completed successfully
		return
	case err := <-errs:
		// Handle errors
	}
}
```