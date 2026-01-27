package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mock database
type mockDB struct {
	healthy bool
	dialect string
}

func (m *mockDB) Ping(ctx context.Context) error {
	if !m.healthy {
		return errors.New("database connection failed")
	}
	return nil
}

func (m *mockDB) Dialect() string {
	return m.dialect
}

// Mock registry
type mockRegistry struct {
	count int
}

func (m *mockRegistry) Count() int {
	return m.count
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", config.Timeout)
	}

	if config.Checkers == nil {
		t.Error("Expected checkers map to be initialized")
	}
}

func TestNewService(t *testing.T) {
	db := &mockDB{healthy: true, dialect: "sqlite"}
	registry := &mockRegistry{count: 5}

	service := NewService(Config{}, db, registry)

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.db != db {
		t.Error("Database not set correctly")
	}

	if service.registry != registry {
		t.Error("Registry not set correctly")
	}
}

func TestNewService_DefaultTimeout(t *testing.T) {
	service := NewService(Config{}, nil, nil)

	if service.config.Timeout != 5*time.Second {
		t.Errorf("Expected default timeout 5s, got %v", service.config.Timeout)
	}
}

func TestService_RegisterChecker(t *testing.T) {
	service := NewService(Config{}, nil, nil)

	service.RegisterChecker("custom", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})

	if len(service.config.Checkers) != 1 {
		t.Error("Expected 1 checker to be registered")
	}

	// Verify the checker exists
	if _, ok := service.config.Checkers["custom"]; !ok {
		t.Error("Expected 'custom' checker to be registered")
	}
}

func TestService_LivenessHandler_Healthy(t *testing.T) {
	db := &mockDB{healthy: true, dialect: "sqlite"}
	registry := &mockRegistry{count: 5}
	service := NewService(Config{Version: "1.0.0"}, db, registry)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	service.LivenessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != StatusHealthy {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}

	if response.Database != "sqlite" {
		t.Errorf("Expected database 'sqlite', got '%s'", response.Database)
	}

	if response.Collections != 5 {
		t.Errorf("Expected collections 5, got %d", response.Collections)
	}

	if response.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", response.Version)
	}
}

func TestService_LivenessHandler_Unhealthy(t *testing.T) {
	db := &mockDB{healthy: false, dialect: "sqlite"}
	service := NewService(Config{}, db, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	service.LivenessHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response HealthResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != StatusUnhealthy {
		t.Errorf("Expected status 'unhealthy', got '%s'", response.Status)
	}
}

func TestService_LivenessHandler_NoDB(t *testing.T) {
	service := NewService(Config{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	service.LivenessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response HealthResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != StatusHealthy {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}
}

func TestService_ReadinessHandler_AllHealthy(t *testing.T) {
	db := &mockDB{healthy: true, dialect: "sqlite"}
	registry := &mockRegistry{count: 5}
	service := NewService(Config{}, db, registry)

	// Add a custom healthy checker
	service.RegisterChecker("external", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy, Message: "OK"}
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	service.ReadinessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ReadinessResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != StatusHealthy {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}

	if len(response.Checks) != 3 { // database, registry, external
		t.Errorf("Expected 3 checks, got %d", len(response.Checks))
	}
}

func TestService_ReadinessHandler_DatabaseUnhealthy(t *testing.T) {
	db := &mockDB{healthy: false, dialect: "sqlite"}
	registry := &mockRegistry{count: 5}
	service := NewService(Config{}, db, registry)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	service.ReadinessHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response ReadinessResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != StatusUnhealthy {
		t.Errorf("Expected status 'unhealthy', got '%s'", response.Status)
	}

	if response.Checks["database"].Status != StatusUnhealthy {
		t.Errorf("Expected database check to be unhealthy")
	}
}

func TestService_ReadinessHandler_CustomCheckerUnhealthy(t *testing.T) {
	db := &mockDB{healthy: true, dialect: "sqlite"}
	service := NewService(Config{}, db, nil)

	service.RegisterChecker("failing", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusUnhealthy, Message: "Service unavailable"}
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	service.ReadinessHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response ReadinessResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != StatusUnhealthy {
		t.Errorf("Expected status 'unhealthy', got '%s'", response.Status)
	}
}

func TestService_ReadinessHandler_DegradedStatus(t *testing.T) {
	service := NewService(Config{}, nil, nil)

	service.RegisterChecker("degraded", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusDegraded, Message: "Partial functionality"}
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	service.ReadinessHandler(w, req)

	// Degraded should still return 200
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ReadinessResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != StatusDegraded {
		t.Errorf("Expected status 'degraded', got '%s'", response.Status)
	}
}

func TestService_IsHealthy(t *testing.T) {
	t.Run("Healthy database", func(t *testing.T) {
		db := &mockDB{healthy: true}
		service := NewService(Config{}, db, nil)

		if !service.IsHealthy(context.Background()) {
			t.Error("Expected IsHealthy to return true")
		}
	})

	t.Run("Unhealthy database", func(t *testing.T) {
		db := &mockDB{healthy: false}
		service := NewService(Config{}, db, nil)

		if service.IsHealthy(context.Background()) {
			t.Error("Expected IsHealthy to return false")
		}
	})

	t.Run("No database", func(t *testing.T) {
		service := NewService(Config{}, nil, nil)

		if !service.IsHealthy(context.Background()) {
			t.Error("Expected IsHealthy to return true when no database")
		}
	})
}

func TestService_IsReady(t *testing.T) {
	t.Run("All healthy", func(t *testing.T) {
		db := &mockDB{healthy: true}
		service := NewService(Config{}, db, nil)

		service.RegisterChecker("test", func(ctx context.Context) CheckResult {
			return CheckResult{Status: StatusHealthy}
		})

		if !service.IsReady(context.Background()) {
			t.Error("Expected IsReady to return true")
		}
	})

	t.Run("Checker unhealthy", func(t *testing.T) {
		db := &mockDB{healthy: true}
		service := NewService(Config{}, db, nil)

		service.RegisterChecker("failing", func(ctx context.Context) CheckResult {
			return CheckResult{Status: StatusUnhealthy}
		})

		if service.IsReady(context.Background()) {
			t.Error("Expected IsReady to return false")
		}
	})

	t.Run("Database unhealthy", func(t *testing.T) {
		db := &mockDB{healthy: false}
		service := NewService(Config{}, db, nil)

		if service.IsReady(context.Background()) {
			t.Error("Expected IsReady to return false")
		}
	})
}

func TestCheckResult(t *testing.T) {
	result := CheckResult{
		Status:  StatusHealthy,
		Message: "Test message",
		Time:    time.Now(),
		Latency: 100 * time.Millisecond,
	}

	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}

	if result.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", result.Message)
	}
}

func TestMemoryChecker(t *testing.T) {
	checker := MemoryChecker(1024)
	result := checker(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}
}

func TestDiskChecker(t *testing.T) {
	checker := DiskChecker("/", 10)
	result := checker(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}
}

func TestExternalServiceChecker(t *testing.T) {
	t.Run("Service available", func(t *testing.T) {
		checker := ExternalServiceChecker("Redis", func(ctx context.Context) error {
			return nil
		})

		result := checker(context.Background())

		if result.Status != StatusHealthy {
			t.Errorf("Expected status healthy, got %s", result.Status)
		}

		if result.Message != "Redis is available" {
			t.Errorf("Expected message 'Redis is available', got '%s'", result.Message)
		}
	})

	t.Run("Service unavailable", func(t *testing.T) {
		checker := ExternalServiceChecker("Redis", func(ctx context.Context) error {
			return errors.New("connection refused")
		})

		result := checker(context.Background())

		if result.Status != StatusUnhealthy {
			t.Errorf("Expected status unhealthy, got %s", result.Status)
		}
	})
}

func TestService_checkDatabase(t *testing.T) {
	t.Run("Healthy", func(t *testing.T) {
		db := &mockDB{healthy: true}
		service := NewService(Config{}, db, nil)

		result := service.checkDatabase(context.Background())

		if result.Status != StatusHealthy {
			t.Errorf("Expected status healthy, got %s", result.Status)
		}

		if result.Latency == 0 {
			t.Error("Expected latency to be set")
		}
	})

	t.Run("Unhealthy", func(t *testing.T) {
		db := &mockDB{healthy: false}
		service := NewService(Config{}, db, nil)

		result := service.checkDatabase(context.Background())

		if result.Status != StatusUnhealthy {
			t.Errorf("Expected status unhealthy, got %s", result.Status)
		}
	})
}

func TestService_checkRegistry(t *testing.T) {
	registry := &mockRegistry{count: 10}
	service := NewService(Config{}, nil, registry)

	result := service.checkRegistry()

	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}
}

func TestStatus_Constants(t *testing.T) {
	if StatusHealthy != "healthy" {
		t.Errorf("Expected 'healthy', got '%s'", StatusHealthy)
	}

	if StatusUnhealthy != "unhealthy" {
		t.Errorf("Expected 'unhealthy', got '%s'", StatusUnhealthy)
	}

	if StatusDegraded != "degraded" {
		t.Errorf("Expected 'degraded', got '%s'", StatusDegraded)
	}
}

func TestService_ConcurrentAccess(t *testing.T) {
	service := NewService(Config{}, nil, nil)

	// Register checkers concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			service.RegisterChecker(string(rune('a'+idx)), func(ctx context.Context) CheckResult {
				return CheckResult{Status: StatusHealthy}
			})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Run readiness check
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	service.ReadinessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}
