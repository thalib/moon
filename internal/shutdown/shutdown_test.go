package shutdown

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if len(config.Signals) == 0 {
		t.Error("Expected default signals to be set")
	}

	if config.Logger == nil {
		t.Error("Expected default logger to be set")
	}
}

func TestNewHandler(t *testing.T) {
	t.Run("With defaults", func(t *testing.T) {
		handler := NewHandler(Config{})

		if handler == nil {
			t.Fatal("NewHandler returned nil")
		}

		if handler.config.Timeout != 30*time.Second {
			t.Errorf("Expected default timeout 30s, got %v", handler.config.Timeout)
		}
	})

	t.Run("With custom config", func(t *testing.T) {
		config := Config{
			Timeout: 10 * time.Second,
			Logger:  &bytes.Buffer{},
		}

		handler := NewHandler(config)

		if handler.config.Timeout != 10*time.Second {
			t.Errorf("Expected timeout 10s, got %v", handler.config.Timeout)
		}
	})
}

func TestHandler_Register(t *testing.T) {
	handler := NewHandler(Config{})

	handler.Register("test", func(ctx context.Context) error {
		return nil
	})

	if len(handler.funcs) != 1 {
		t.Errorf("Expected 1 function, got %d", len(handler.funcs))
	}

	if handler.funcs[0].name != "test" {
		t.Errorf("Expected name 'test', got '%s'", handler.funcs[0].name)
	}
}

func TestHandler_RegisterMultiple(t *testing.T) {
	handler := NewHandler(Config{})

	handler.Register("first", func(ctx context.Context) error { return nil })
	handler.Register("second", func(ctx context.Context) error { return nil })
	handler.Register("third", func(ctx context.Context) error { return nil })

	if len(handler.funcs) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(handler.funcs))
	}
}

func TestHandler_RegisterCloser(t *testing.T) {
	handler := NewHandler(Config{})

	closed := false
	closer := &testCloser{onClose: func() error {
		closed = true
		return nil
	}}

	handler.RegisterCloser("db", closer)

	if len(handler.funcs) != 1 {
		t.Error("Expected 1 function registered")
	}

	// Execute the shutdown function
	handler.funcs[0].fn(context.Background())

	if !closed {
		t.Error("Closer should have been called")
	}
}

type testCloser struct {
	onClose func() error
}

func (tc *testCloser) Close() error {
	if tc.onClose != nil {
		return tc.onClose()
	}
	return nil
}

func TestHandler_RegisterServer(t *testing.T) {
	handler := NewHandler(Config{})

	shutdown := false
	server := &testServer{onShutdown: func(ctx context.Context) error {
		shutdown = true
		return nil
	}}

	handler.RegisterServer("http", server)

	if len(handler.funcs) != 1 {
		t.Error("Expected 1 function registered")
	}

	// Execute the shutdown function
	handler.funcs[0].fn(context.Background())

	if !shutdown {
		t.Error("Server should have been shutdown")
	}
}

type testServer struct {
	onShutdown func(ctx context.Context) error
}

func (ts *testServer) Shutdown(ctx context.Context) error {
	if ts.onShutdown != nil {
		return ts.onShutdown(ctx)
	}
	return nil
}

func TestHandler_Trigger(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(Config{
		Timeout: 1 * time.Second,
		Logger:  &buf,
	})

	var callOrder []string
	var mu sync.Mutex

	handler.Register("first", func(ctx context.Context) error {
		mu.Lock()
		callOrder = append(callOrder, "first")
		mu.Unlock()
		return nil
	})

	handler.Register("second", func(ctx context.Context) error {
		mu.Lock()
		callOrder = append(callOrder, "second")
		mu.Unlock()
		return nil
	})

	// Trigger shutdown
	handler.Trigger()

	// Wait for shutdown to complete
	select {
	case <-handler.doneChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}

	// Verify LIFO order
	mu.Lock()
	defer mu.Unlock()

	if len(callOrder) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(callOrder))
	}

	if callOrder[0] != "second" {
		t.Errorf("Expected 'second' first (LIFO), got '%s'", callOrder[0])
	}

	if callOrder[1] != "first" {
		t.Errorf("Expected 'first' second (LIFO), got '%s'", callOrder[1])
	}
}

func TestHandler_ShutdownError(t *testing.T) {
	var buf bytes.Buffer
	var capturedErr error

	handler := NewHandler(Config{
		Timeout: 1 * time.Second,
		Logger:  &buf,
		OnShutdownComplete: func(err error) {
			capturedErr = err
		},
	})

	expectedErr := errors.New("shutdown error")
	handler.Register("failing", func(ctx context.Context) error {
		return expectedErr
	})

	handler.Trigger()

	select {
	case <-handler.doneChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}

	if capturedErr == nil {
		t.Error("Expected error to be captured")
	}
}

func TestHandler_ShutdownTimeout(t *testing.T) {
	var buf bytes.Buffer
	var capturedErr error

	handler := NewHandler(Config{
		Timeout: 100 * time.Millisecond,
		Logger:  &buf,
		OnShutdownComplete: func(err error) {
			capturedErr = err
		},
	})

	handler.Register("slow", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	})

	handler.Trigger()

	select {
	case <-handler.doneChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}

	if capturedErr == nil {
		t.Error("Expected timeout error")
	}
}

func TestHandler_Callbacks(t *testing.T) {
	var buf bytes.Buffer

	startCalled := false
	completeCalled := false

	handler := NewHandler(Config{
		Timeout: 1 * time.Second,
		Logger:  &buf,
		OnShutdownStart: func() {
			startCalled = true
		},
		OnShutdownComplete: func(err error) {
			completeCalled = true
		},
	})

	handler.Trigger()

	select {
	case <-handler.doneChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}

	if !startCalled {
		t.Error("OnShutdownStart should have been called")
	}

	if !completeCalled {
		t.Error("OnShutdownComplete should have been called")
	}
}

func TestHandler_DoubleTrigger(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(Config{
		Timeout: 1 * time.Second,
		Logger:  &buf,
	})

	callCount := int32(0)
	handler.Register("test", func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	// Trigger twice
	handler.Trigger()
	handler.Trigger()

	select {
	case <-handler.doneChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("Expected shutdown to run only once, got %d", callCount)
	}
}

func TestHandler_StartOnce(t *testing.T) {
	handler := NewHandler(Config{
		Timeout: 1 * time.Second,
		Signals: []os.Signal{syscall.SIGUSR1}, // Use a signal we won't receive
	})

	// Start twice should not block
	go handler.Start()
	go handler.Start()

	// Give time for both goroutines to start
	time.Sleep(100 * time.Millisecond)

	// Both should have started, but only one is actually listening
	// The second Start() should have returned immediately

	// Trigger shutdown
	handler.Trigger()

	select {
	case <-handler.doneChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}
}

func TestGracefulServer(t *testing.T) {
	var buf bytes.Buffer

	startCalled := false
	shutdownCalled := false

	server := &testServer{
		onShutdown: func(ctx context.Context) error {
			shutdownCalled = true
			return nil
		},
	}

	startFunc := func() error {
		startCalled = true
		// Simulate server running until shutdown
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	gs := NewGracefulServer(server, startFunc, Config{
		Timeout: 1 * time.Second,
		Logger:  &buf,
	})

	// Run in goroutine since it blocks
	done := make(chan error)
	go func() {
		done <- gs.Run()
	}()

	// Give time for server to start
	time.Sleep(100 * time.Millisecond)

	// Trigger shutdown
	gs.Shutdown()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete in time")
	}

	if !startCalled {
		t.Error("Start function should have been called")
	}

	if !shutdownCalled {
		t.Error("Server shutdown should have been called")
	}
}

func TestGracefulServer_Register(t *testing.T) {
	server := &testServer{}
	gs := NewGracefulServer(server, func() error { return nil }, Config{})

	gs.Register("custom", func(ctx context.Context) error {
		return nil
	})

	// Check that function was registered
	if len(gs.shutdownHandler.funcs) != 2 { // server + custom
		t.Errorf("Expected 2 functions, got %d", len(gs.shutdownHandler.funcs))
	}
}

func TestGracefulServer_RegisterCloser(t *testing.T) {
	server := &testServer{}
	gs := NewGracefulServer(server, func() error { return nil }, Config{})

	closer := &testCloser{onClose: func() error {
		return nil
	}}

	gs.RegisterCloser("db", closer)

	// Check that function was registered
	if len(gs.shutdownHandler.funcs) != 2 { // server + db
		t.Errorf("Expected 2 functions, got %d", len(gs.shutdownHandler.funcs))
	}
}

func TestHandler_ConcurrentRegister(t *testing.T) {
	handler := NewHandler(Config{})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			handler.Register(
				string(rune('a'+idx%26)),
				func(ctx context.Context) error { return nil },
			)
		}(i)
	}

	wg.Wait()

	if len(handler.funcs) != 100 {
		t.Errorf("Expected 100 functions, got %d", len(handler.funcs))
	}
}

func TestFormatString(t *testing.T) {
	result := formatString("hello %s", "world")
	if result != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", result)
	}

	result = formatString("no args")
	if result != "no args" {
		t.Errorf("Expected 'no args', got '%s'", result)
	}
}

// Suppress log output during tests
func init() {
	// Redirect log output to discard during tests
	// This is only for cleaner test output
}

// mockWriter implements io.Writer for testing
type mockWriter struct {
	written []byte
	mu      sync.Mutex
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written, p...)
	return len(p), nil
}

var _ io.Writer = (*mockWriter)(nil)
