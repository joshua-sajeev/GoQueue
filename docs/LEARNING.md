# Learnings from GoQueue project
## 2025-12-06
### Mocking functions
```go
var envProcess = envconfig.Process
```
`envconfig.Process` is actually a function. But to simulate errors we can assign it to a var and then play with it like 
```go
envProcess = func(ctx context.Context, i any, mus ...envconfig.Mutator) error {
	return fmt.Errorf("mock envconfig error")
}
```

### context.Context
“Pass context.Context as the first parameter to any function that might block or take a long time.”
- Without ctx, your DB operations keep running even after request is gone → memory leaks + connection exhaustion.
- If your repo methods don’t accept ctx, your service cannot control cancellation or deadlines.

## 2025-12-15
### Postgres Connection
Postgres won't ask for a password when connecting locally inside the container, and Postgres because configured to trust local connections.
