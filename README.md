# go-graceful
go-graceful pkg listens to signals or context cancelation to initiate shutdown.

```go 
<-graceful.Shutdown(ctx, timeout, map[string]graceful.Operation{
	"kafka":       ShutdownKafka,
	"database":    ShutdownDB,
	"http-server": ShutdownHTTP,
})
```

# TODO:
What can be done to make this pkg more public friendly:
- Logs aren't really necessary: You can log inside operations. You can replace with DEBUG level logs. or find ways (ex: tags) to toggle logs on and off completely etc.
- You better return an error, or err chan or even wrapped ctx.
