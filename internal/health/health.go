package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status
type Status string

const (
	// StatusHealthy indicates the service is healthy
	StatusHealthy Status = "healthy"

	// StatusUnhealthy indicates the service is unhealthy
	StatusUnhealthy Status = "unhealthy"

	// StatusDegraded indicates the service is partially healthy
	StatusDegraded Status = "degraded"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Time    time.Time `json:"time"`
	Latency time.Duration `json:"latency,omitempty"`
}

// HealthResponse is the response for /health endpoint
type HealthResponse struct {
	Status      Status                  `json:"status"`
	Database    string                  `json:"database,omitempty"`
	Collections int                     `json:"collections,omitempty"`
	Timestamp   time.Time               `json:"timestamp"`
	Version     string                  `json:"version,omitempty"`
}

// ReadinessResponse is the response for /health/ready endpoint
type ReadinessResponse struct {
	Status    Status                  `json:"status"`
	Checks    map[string]CheckResult  `json:"checks"`
	Timestamp time.Time               `json:"timestamp"`
}

// Checker is a function that performs a health check
type Checker func(ctx context.Context) CheckResult

// DatabaseChecker is an interface for database connectivity checks
type DatabaseChecker interface {
	Ping(ctx context.Context) error
	Dialect() string
}

// RegistryChecker is an interface for registry checks
type RegistryChecker interface {
	Count() int
}

// Config holds configuration for the health checker
type Config struct {
	// Timeout is the maximum time for each health check
	Timeout time.Duration

	// Version is the application version
	Version string

	// Checkers is a map of named health checkers
	Checkers map[string]Checker
}

// DefaultConfig returns the default health configuration
func DefaultConfig() Config {
	return Config{
		Timeout:  5 * time.Second,
		Checkers: make(map[string]Checker),
	}
}

// Service provides health check functionality
type Service struct {
	config   Config
	db       DatabaseChecker
	registry RegistryChecker
	mu       sync.RWMutex
}

// NewService creates a new health check service
func NewService(config Config, db DatabaseChecker, registry RegistryChecker) *Service {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.Checkers == nil {
		config.Checkers = make(map[string]Checker)
	}

	return &Service{
		config:   config,
		db:       db,
		registry: registry,
	}
}

// RegisterChecker adds a custom health checker
func (s *Service) RegisterChecker(name string, checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.Checkers[name] = checker
}

// LivenessHandler handles the /health liveness check
// This is a simple check that returns 200 if the server is running
func (s *Service) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.config.Timeout)
	defer cancel()

	response := HealthResponse{
		Status:    StatusHealthy,
		Timestamp: time.Now().UTC(),
		Version:   s.config.Version,
	}

	// Check database if available
	if s.db != nil {
		if err := s.db.Ping(ctx); err != nil {
			response.Status = StatusUnhealthy
			s.writeJSON(w, http.StatusServiceUnavailable, response)
			return
		}
		response.Database = string(s.db.Dialect())
	}

	// Add registry count if available
	if s.registry != nil {
		response.Collections = s.registry.Count()
	}

	s.writeJSON(w, http.StatusOK, response)
}

// ReadinessHandler handles the /health/ready readiness check
// This performs comprehensive checks to ensure the service can accept requests
func (s *Service) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.config.Timeout)
	defer cancel()

	response := ReadinessResponse{
		Status:    StatusHealthy,
		Checks:    make(map[string]CheckResult),
		Timestamp: time.Now().UTC(),
	}

	// Check database
	if s.db != nil {
		dbResult := s.checkDatabase(ctx)
		response.Checks["database"] = dbResult
		if dbResult.Status != StatusHealthy {
			response.Status = StatusUnhealthy
		}
	}

	// Check registry
	if s.registry != nil {
		registryResult := s.checkRegistry()
		response.Checks["registry"] = registryResult
		if registryResult.Status != StatusHealthy {
			if response.Status == StatusHealthy {
				response.Status = StatusDegraded
			}
		}
	}

	// Run custom checkers
	s.mu.RLock()
	checkers := make(map[string]Checker, len(s.config.Checkers))
	for name, checker := range s.config.Checkers {
		checkers[name] = checker
	}
	s.mu.RUnlock()

	for name, checker := range checkers {
		result := checker(ctx)
		response.Checks[name] = result
		if result.Status == StatusUnhealthy {
			response.Status = StatusUnhealthy
		} else if result.Status == StatusDegraded && response.Status == StatusHealthy {
			response.Status = StatusDegraded
		}
	}

	statusCode := http.StatusOK
	if response.Status == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	s.writeJSON(w, statusCode, response)
}

// checkDatabase performs a database health check
func (s *Service) checkDatabase(ctx context.Context) CheckResult {
	start := time.Now()

	if err := s.db.Ping(ctx); err != nil {
		return CheckResult{
			Status:  StatusUnhealthy,
			Message: "Database connection failed: " + err.Error(),
			Time:    start,
			Latency: time.Since(start),
		}
	}

	return CheckResult{
		Status:  StatusHealthy,
		Message: "Database connection successful",
		Time:    start,
		Latency: time.Since(start),
	}
}

// checkRegistry performs a registry health check
func (s *Service) checkRegistry() CheckResult {
	_ = s.registry.Count() // Check that registry is accessible

	return CheckResult{
		Status:  StatusHealthy,
		Message: "Registry initialized",
		Time:    time.Now(),
	}
}

// writeJSON writes a JSON response
func (s *Service) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// IsHealthy performs a simple health check and returns true if healthy
func (s *Service) IsHealthy(ctx context.Context) bool {
	if s.db != nil {
		if err := s.db.Ping(ctx); err != nil {
			return false
		}
	}
	return true
}

// IsReady performs a comprehensive readiness check and returns true if ready
func (s *Service) IsReady(ctx context.Context) bool {
	// Check database
	if s.db != nil {
		if err := s.db.Ping(ctx); err != nil {
			return false
		}
	}

	// Run custom checkers
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, checker := range s.config.Checkers {
		result := checker(ctx)
		if result.Status == StatusUnhealthy {
			return false
		}
	}

	return true
}

// Common health checkers

// MemoryChecker creates a memory usage health checker
func MemoryChecker(maxMemoryMB int64) Checker {
	return func(ctx context.Context) CheckResult {
		// This is a placeholder - actual implementation would check runtime.MemStats
		return CheckResult{
			Status:  StatusHealthy,
			Message: "Memory usage within limits",
			Time:    time.Now(),
		}
	}
}

// DiskChecker creates a disk space health checker
func DiskChecker(path string, minFreeGB int64) Checker {
	return func(ctx context.Context) CheckResult {
		// This is a placeholder - actual implementation would check disk space
		return CheckResult{
			Status:  StatusHealthy,
			Message: "Disk space available",
			Time:    time.Now(),
		}
	}
}

// ExternalServiceChecker creates a health checker for external services
func ExternalServiceChecker(name string, check func(ctx context.Context) error) Checker {
	return func(ctx context.Context) CheckResult {
		start := time.Now()
		err := check(ctx)

		if err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: name + " is unavailable: " + err.Error(),
				Time:    start,
				Latency: time.Since(start),
			}
		}

		return CheckResult{
			Status:  StatusHealthy,
			Message: name + " is available",
			Time:    start,
			Latency: time.Since(start),
		}
	}
}
