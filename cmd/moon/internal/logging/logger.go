// Package logging provides structured logging with zerolog.
// It supports JSON and console formats, log levels, file output,
// request ID tracking, and automatic masking of sensitive fields.
package logging

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Level represents logging levels
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level Level

	// Format is the output format (json or console)
	Format string

	// Output is the writer for logs (default: os.Stdout)
	Output io.Writer

	// FilePath is the path to the log file (if specified, Output is ignored)
	FilePath string

	// ServiceName is the name of the service
	ServiceName string

	// Version is the version of the service
	Version string

	// SlowQueryThreshold is the duration after which a query is considered slow
	SlowQueryThreshold time.Duration

	// SensitiveFields are field names that should be masked in logs
	SensitiveFields []string
}

// Logger wraps zerolog for structured logging
type Logger struct {
	logger             zerolog.Logger
	config             LoggerConfig
	sensitiveFields    map[string]bool
}

// NewLogger creates a new structured logger
func NewLogger(config LoggerConfig) *Logger {
	var output io.Writer

	// If FilePath is specified, open/create log file
	if config.FilePath != "" {
		// Ensure log directory exists
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			// If we can't create directory, fall back to stdout
			fmt.Fprintf(os.Stderr, "Failed to create log directory %s: %v\n", dir, err)
			output = os.Stdout
		} else {
			file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				// If we can't open file, fall back to stdout
				fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", config.FilePath, err)
				output = os.Stdout
			} else {
				output = file
			}
		}
	} else if config.Output != nil {
		output = config.Output
	} else {
		output = os.Stdout
	}

	if config.Level == "" {
		config.Level = LevelInfo
	}

	if config.SlowQueryThreshold == 0 {
		config.SlowQueryThreshold = 500 * time.Millisecond
	}

	// Set zerolog level
	var zeroLevel zerolog.Level
	switch config.Level {
	case LevelDebug:
		zeroLevel = zerolog.DebugLevel
	case LevelInfo:
		zeroLevel = zerolog.InfoLevel
	case LevelWarn:
		zeroLevel = zerolog.WarnLevel
	case LevelError:
		zeroLevel = zerolog.ErrorLevel
	default:
		zeroLevel = zerolog.InfoLevel
	}

	var logger zerolog.Logger

	// Configure output format
	if config.Format == "console" {
		consoleOutput := zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(consoleOutput).Level(zeroLevel).With().Timestamp().Logger()
	} else {
		// Default to JSON format
		logger = zerolog.New(output).Level(zeroLevel).With().Timestamp().Logger()
	}

	// Add service context
	if config.ServiceName != "" {
		logger = logger.With().Str("service", config.ServiceName).Logger()
	}
	if config.Version != "" {
		logger = logger.With().Str("version", config.Version).Logger()
	}

	// Build sensitive fields map (case-insensitive)
	sensitiveFields := make(map[string]bool)
	for _, field := range config.SensitiveFields {
		sensitiveFields[strings.ToLower(field)] = true
	}

	// Add default sensitive fields (lowercase for case-insensitive matching)
	defaultSensitive := []string{"password", "token", "secret", "api_key", "apikey", "authorization"}
	for _, field := range defaultSensitive {
		sensitiveFields[field] = true
	}

	return &Logger{
		logger:          logger,
		config:          config,
		sensitiveFields: sensitiveFields,
	}
}

// WithContext returns a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *Logger {
	newLogger := *l
	
	// Add request ID if present
	if requestID := GetRequestID(ctx); requestID != "" {
		newLogger.logger = l.logger.With().Str("request_id", requestID).Logger()
	}

	return &newLogger
}

// WithField returns a logger with an additional field
func (l *Logger) WithField(key string, value any) *Logger {
	newLogger := *l
	newLogger.logger = l.logger.With().Interface(key, l.maskSensitive(key, value)).Logger()
	return &newLogger
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]any) *Logger {
	newLogger := *l
	ctx := l.logger.With()
	for key, value := range fields {
		ctx = ctx.Interface(key, l.maskSensitive(key, value))
	}
	newLogger.logger = ctx.Logger()
	return &newLogger
}

// maskSensitive masks sensitive field values (case-insensitive)
func (l *Logger) maskSensitive(key string, value any) any {
	if l.sensitiveFields[strings.ToLower(key)] {
		return "***REDACTED***"
	}
	return value
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...any) {
	l.logger.Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...any) {
	l.logger.Info().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...any) {
	l.logger.Warn().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...any) {
	l.logger.Error().Msgf(format, args...)
}

// ErrorWithErr logs an error with the error object
func (l *Logger) ErrorWithErr(msg string, err error) {
	l.logger.Error().Err(err).Msg(msg)
}

// LogSlowQuery logs a slow query warning
func (l *Logger) LogSlowQuery(query string, duration time.Duration, args ...any) {
	if duration >= l.config.SlowQueryThreshold {
		l.logger.Warn().
			Str("query", query).
			Dur("duration", duration).
			Interface("args", args).
			Msg("Slow query detected")
	}
}

// Context key for request ID
type contextKey string

const requestIDKey contextKey = "request_id"

// SetRequestID sets the request ID in the context
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID gets the request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// RequestLoggerConfig holds configuration for request logging middleware
type RequestLoggerConfig struct {
	Logger *Logger

	// SkipPaths are paths that should not be logged
	SkipPaths []string

	// LogBody logs request and response bodies (use with caution)
	LogBody bool

	// LogHeaders logs request headers
	LogHeaders bool
}

// RequestLogger is middleware for logging HTTP requests
type RequestLogger struct {
	config    RequestLoggerConfig
	skipPaths map[string]bool
}

// NewRequestLogger creates a new request logging middleware
func NewRequestLogger(config RequestLoggerConfig) *RequestLogger {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return &RequestLogger{
		config:    config,
		skipPaths: skipPaths,
	}
}

// Middleware returns the HTTP middleware function
func (rl *RequestLogger) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for certain paths
		if rl.skipPaths[r.URL.Path] {
			next(w, r)
			return
		}

		start := time.Now()

		// Generate or get request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to response header
		w.Header().Set("X-Request-ID", requestID)

		// Add request ID to context
		ctx := SetRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		// Create response writer wrapper
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			bytesWritten:   0,
		}

		// Log request start (debug level)
		logger := rl.config.Logger.WithContext(ctx)

		if rl.config.Logger.config.Level == LevelDebug {
			debugFields := map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
				"query":  r.URL.RawQuery,
			}

			if rl.config.LogHeaders {
				headers := make(map[string]string)
				for key, values := range r.Header {
					// Mask sensitive headers
					if rl.config.Logger.sensitiveFields[key] {
						headers[key] = "***REDACTED***"
					} else if len(values) > 0 {
						headers[key] = values[0]
					}
				}
				debugFields["headers"] = headers
			}

			logger.WithFields(debugFields).Debug("Request started")
		}

		// Call next handler
		next(rw, r)

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		event := rl.config.Logger.logger.Info()
		if rw.statusCode >= 500 {
			event = rl.config.Logger.logger.Error()
		} else if rw.statusCode >= 400 {
			event = rl.config.Logger.logger.Warn()
		}

		event.
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", rw.statusCode).
			Dur("duration", duration).
			Int("bytes", rw.bytesWritten).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Msg("Request completed")
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Global logger instance
var globalLogger *Logger

// Init initializes the global logger
func Init(config LoggerConfig) {
	globalLogger = NewLogger(config)
}

// GetLogger returns the global logger
func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default config
		globalLogger = NewLogger(LoggerConfig{
			Level:  LevelInfo,
			Format: "json",
		})
	}
	return globalLogger
}

// Debug logs a debug message using the global logger
func Debug(msg string) {
	GetLogger().Debug(msg)
}

// Debugf logs a formatted debug message using the global logger
func Debugf(format string, args ...any) {
	GetLogger().Debugf(format, args...)
}

// Info logs an info message using the global logger
func Info(msg string) {
	GetLogger().Info(msg)
}

// Infof logs a formatted info message using the global logger
func Infof(format string, args ...any) {
	GetLogger().Infof(format, args...)
}

// Warn logs a warning message using the global logger
func Warn(msg string) {
	GetLogger().Warn(msg)
}

// Warnf logs a formatted warning message using the global logger
func Warnf(format string, args ...any) {
	GetLogger().Warnf(format, args...)
}

// Error logs an error message using the global logger
func Error(msg string) {
	GetLogger().Error(msg)
}

// Errorf logs a formatted error message using the global logger
func Errorf(format string, args ...any) {
	GetLogger().Errorf(format, args...)
}

// ErrorWithErr logs an error with the error object using the global logger
func ErrorWithErr(msg string, err error) {
	GetLogger().ErrorWithErr(msg, err)
}
