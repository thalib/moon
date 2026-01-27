package shutdown

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownFunc is a function that will be called during shutdown
type ShutdownFunc func(ctx context.Context) error

// Config holds configuration for the shutdown handler
type Config struct {
	// Timeout is the maximum time to wait for shutdown completion
	Timeout time.Duration

	// Signals is the list of OS signals to listen for
	Signals []os.Signal

	// OnShutdownStart is called when shutdown begins
	OnShutdownStart func()

	// OnShutdownComplete is called when shutdown completes
	OnShutdownComplete func(err error)

	// Logger is used for shutdown logging (default: os.Stdout)
	Logger io.Writer
}

// DefaultConfig returns the default shutdown configuration
func DefaultConfig() Config {
	return Config{
		Timeout: 30 * time.Second,
		Signals: []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT},
		Logger:  os.Stdout,
	}
}

// Handler manages graceful shutdown of services
type Handler struct {
	config       Config
	funcs        []namedShutdownFunc
	mu           sync.Mutex
	shutdownChan chan os.Signal
	doneChan     chan struct{}
	started      bool
	shuttingDown bool
}

// namedShutdownFunc is a shutdown function with a name for logging
type namedShutdownFunc struct {
	name string
	fn   ShutdownFunc
}

// NewHandler creates a new shutdown handler
func NewHandler(config Config) *Handler {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Signals == nil {
		config.Signals = []os.Signal{os.Interrupt, syscall.SIGTERM}
	}
	if config.Logger == nil {
		config.Logger = os.Stdout
	}

	return &Handler{
		config:       config,
		funcs:        make([]namedShutdownFunc, 0),
		shutdownChan: make(chan os.Signal, 1),
		doneChan:     make(chan struct{}),
	}
}

// Register adds a shutdown function to be called during shutdown
// Functions are called in LIFO order (last registered, first called)
func (h *Handler) Register(name string, fn ShutdownFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.funcs = append(h.funcs, namedShutdownFunc{
		name: name,
		fn:   fn,
	})
}

// RegisterServer is a convenience method to register an HTTP server for shutdown
func (h *Handler) RegisterServer(name string, server HTTPServer) {
	h.Register(name, func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})
}

// RegisterCloser is a convenience method to register an io.Closer for shutdown
func (h *Handler) RegisterCloser(name string, closer io.Closer) {
	h.Register(name, func(ctx context.Context) error {
		return closer.Close()
	})
}

// HTTPServer is an interface for HTTP servers that support graceful shutdown
type HTTPServer interface {
	Shutdown(ctx context.Context) error
}

// Start begins listening for shutdown signals
// This method should be called in a goroutine
func (h *Handler) Start() {
	h.mu.Lock()
	if h.started {
		h.mu.Unlock()
		return
	}
	h.started = true
	h.mu.Unlock()

	signal.Notify(h.shutdownChan, h.config.Signals...)

	<-h.shutdownChan
	h.performShutdown()
}

// Wait blocks until shutdown is complete
func (h *Handler) Wait() {
	<-h.doneChan
}

// Trigger initiates shutdown programmatically (useful for testing)
func (h *Handler) Trigger() {
	h.mu.Lock()
	if h.shuttingDown {
		h.mu.Unlock()
		return
	}
	h.shuttingDown = true
	h.mu.Unlock()

	// If not started, start and immediately shutdown
	if !h.started {
		go h.performShutdown()
		return
	}

	// Send signal to shutdown channel
	select {
	case h.shutdownChan <- syscall.SIGTERM:
	default:
	}
}

// performShutdown executes all registered shutdown functions
func (h *Handler) performShutdown() {
	defer close(h.doneChan)

	h.log("Shutdown signal received")

	if h.config.OnShutdownStart != nil {
		h.config.OnShutdownStart()
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout)
	defer cancel()

	var shutdownErr error

	// Execute shutdown functions in reverse order (LIFO)
	h.mu.Lock()
	funcs := make([]namedShutdownFunc, len(h.funcs))
	copy(funcs, h.funcs)
	h.mu.Unlock()

	for i := len(funcs) - 1; i >= 0; i-- {
		f := funcs[i]
		h.log("Shutting down: %s", f.name)

		start := time.Now()
		if err := f.fn(ctx); err != nil {
			h.log("Error shutting down %s: %v", f.name, err)
			if shutdownErr == nil {
				shutdownErr = err
			}
		} else {
			h.log("Shut down %s successfully (took %v)", f.name, time.Since(start))
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		h.log("Shutdown timeout exceeded")
		if shutdownErr == nil {
			shutdownErr = ctx.Err()
		}
	}

	if h.config.OnShutdownComplete != nil {
		h.config.OnShutdownComplete(shutdownErr)
	}

	h.log("Shutdown complete")
}

// log writes a log message
func (h *Handler) log(format string, args ...any) {
	msg := format
	if len(args) > 0 {
		msg = formatString(format, args...)
	}
	log.SetOutput(h.config.Logger)
	log.Printf("[SHUTDOWN] %s", msg)
}

// formatString formats a string with arguments
func formatString(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// GracefulServer wraps an HTTP server with graceful shutdown support
type GracefulServer struct {
	server          HTTPServer
	shutdownHandler *Handler
	startFunc       func() error
}

// NewGracefulServer creates a new graceful server wrapper
func NewGracefulServer(server HTTPServer, startFunc func() error, config Config) *GracefulServer {
	handler := NewHandler(config)
	handler.RegisterServer("http-server", server)

	return &GracefulServer{
		server:          server,
		shutdownHandler: handler,
		startFunc:       startFunc,
	}
}

// Register adds a shutdown function to be called during shutdown
func (gs *GracefulServer) Register(name string, fn ShutdownFunc) {
	gs.shutdownHandler.Register(name, fn)
}

// RegisterCloser adds an io.Closer to be closed during shutdown
func (gs *GracefulServer) RegisterCloser(name string, closer io.Closer) {
	gs.shutdownHandler.RegisterCloser(name, closer)
}

// Run starts the server and handles graceful shutdown
func (gs *GracefulServer) Run() error {
	// Start listening for shutdown signals
	go gs.shutdownHandler.Start()

	// Start the server
	serverErr := make(chan error, 1)
	go func() {
		err := gs.startFunc()
		// http.Server.Shutdown() returns http.ErrServerClosed when closed gracefully
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for shutdown to complete
	gs.shutdownHandler.Wait()

	// Return any server error
	select {
	case err := <-serverErr:
		return err
	default:
		return nil
	}
}

// Shutdown triggers shutdown programmatically
func (gs *GracefulServer) Shutdown() {
	gs.shutdownHandler.Trigger()
}
