package app

import (
	"context"
	"time"
)

// Mock implementations
type mockServer struct {
	started  bool
	shutdown bool
}

func (m *mockServer) ListenAndServe() error {
	m.started = true
	return nil
}

func (m *mockServer) Shutdown(ctx context.Context) error {
	m.shutdown = true
	return nil
}

type mockServerWithShutdownError struct {
	shutdownErr    error
	shutdownCalled bool
}

func (m *mockServerWithShutdownError) ListenAndServe() error {
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (m *mockServerWithShutdownError) Shutdown(ctx context.Context) error {
	m.shutdownCalled = true
	return m.shutdownErr
}
