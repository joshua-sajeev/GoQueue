# Learnings from GoQueue project
## 2025-12-06
```go
var envProcess = envconfig.Process
```
`envconfig.Process` is actually a function. But to simulate errors we can assign it to a var and then play with it like 
```go
envProcess = func(ctx context.Context, i any, mus ...envconfig.Mutator) error {
	return fmt.Errorf("mock envconfig error")
}
```