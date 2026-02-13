package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCORSEndpointConfig_Matches tests pattern matching for CORS endpoints
func TestCORSEndpointConfig_Matches(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    CORSEndpointConfig
		path        string
		wantMatch   bool
		wantScore   int
		description string
	}{
		// Exact matches
		{
			name: "exact_match_health",
			endpoint: CORSEndpointConfig{
				Path:        "/health",
				PatternType: "exact",
			},
			path:        "/health",
			wantMatch:   true,
			wantScore:   1007, // 1000 + len("/health")
			description: "Exact match for /health",
		},
		{
			name: "exact_no_match_health_status",
			endpoint: CORSEndpointConfig{
				Path:        "/health",
				PatternType: "exact",
			},
			path:        "/health/status",
			wantMatch:   false,
			wantScore:   0,
			description: "Exact match should not match /health/status",
		},
		// Prefix matches
		{
			name: "prefix_match_doc",
			endpoint: CORSEndpointConfig{
				Path:        "/doc/*",
				PatternType: "prefix",
			},
			path:        "/doc",
			wantMatch:   true,
			wantScore:   504, // 500 + len("/doc")
			description: "Prefix match for /doc",
		},
		{
			name: "prefix_match_doc_api",
			endpoint: CORSEndpointConfig{
				Path:        "/doc/*",
				PatternType: "prefix",
			},
			path:        "/doc/api",
			wantMatch:   true,
			wantScore:   504,
			description: "Prefix match for /doc/api",
		},
		{
			name: "prefix_match_doc_llms_md",
			endpoint: CORSEndpointConfig{
				Path:        "/doc/*",
				PatternType: "prefix",
			},
			path:        "/doc/llms.md",
			wantMatch:   true,
			wantScore:   504,
			description: "Prefix match for /doc/llms.md",
		},
		{
			name: "prefix_no_match_different_path",
			endpoint: CORSEndpointConfig{
				Path:        "/doc/*",
				PatternType: "prefix",
			},
			path:        "/api/data",
			wantMatch:   false,
			wantScore:   0,
			description: "Prefix should not match different path",
		},
		// Suffix matches
		{
			name: "suffix_match_json",
			endpoint: CORSEndpointConfig{
				Path:        "*.json",
				PatternType: "suffix",
			},
			path:        "/data/users.json",
			wantMatch:   true,
			wantScore:   305, // 300 + len(".json")
			description: "Suffix match for .json files",
		},
		{
			name: "suffix_match_reports_json",
			endpoint: CORSEndpointConfig{
				Path:        "*.json",
				PatternType: "suffix",
			},
			path:        "/reports/summary.json",
			wantMatch:   true,
			wantScore:   305,
			description: "Suffix match for .json in different path",
		},
		{
			name: "suffix_no_match_txt",
			endpoint: CORSEndpointConfig{
				Path:        "*.json",
				PatternType: "suffix",
			},
			path:        "/doc/llms-full.txt",
			wantMatch:   false,
			wantScore:   0,
			description: "Suffix should not match different extension",
		},
		// Contains matches
		{
			name: "contains_match_public",
			endpoint: CORSEndpointConfig{
				Path:        "/public/",
				PatternType: "contains",
			},
			path:        "/api/public/data",
			wantMatch:   true,
			wantScore:   108, // 100 + len("/public/")
			description: "Contains match for /public/ in path",
		},
		{
			name: "contains_match_public_reports",
			endpoint: CORSEndpointConfig{
				Path:        "/public/",
				PatternType: "contains",
			},
			path:        "/public/reports/summary",
			wantMatch:   true,
			wantScore:   108,
			description: "Contains match for /public/ at start",
		},
		{
			name: "contains_no_match_private",
			endpoint: CORSEndpointConfig{
				Path:        "/public/",
				PatternType: "contains",
			},
			path:        "/private/data",
			wantMatch:   false,
			wantScore:   0,
			description: "Contains should not match different substring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch, gotScore := tt.endpoint.Matches(tt.path)
			if gotMatch != tt.wantMatch {
				t.Errorf("Matches() match = %v, want %v for %s", gotMatch, tt.wantMatch, tt.description)
			}
			if gotScore != tt.wantScore {
				t.Errorf("Matches() score = %v, want %v for %s", gotScore, tt.wantScore, tt.description)
			}
		})
	}
}

// TestCORSMiddleware_MatchEndpoint tests endpoint matching priority
func TestCORSMiddleware_MatchEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		endpoints      []CORSEndpointConfig
		path           string
		wantPath       string
		wantType       string
		wantBypassAuth bool
		description    string
	}{
		{
			name: "exact_over_prefix",
			endpoints: []CORSEndpointConfig{
				{
					Path:        "/health/*",
					PatternType: "prefix",
					BypassAuth:  false,
				},
				{
					Path:        "/health",
					PatternType: "exact",
					BypassAuth:  true,
				},
			},
			path:           "/health",
			wantPath:       "/health",
			wantType:       "exact",
			wantBypassAuth: true,
			description:    "Exact match should take priority over prefix",
		},
		{
			name: "longest_prefix",
			endpoints: []CORSEndpointConfig{
				{
					Path:        "/api/*",
					PatternType: "prefix",
					BypassAuth:  false,
				},
				{
					Path:        "/api/v1/public/*",
					PatternType: "prefix",
					BypassAuth:  true,
				},
			},
			path:           "/api/v1/public/data",
			wantPath:       "/api/v1/public/*",
			wantType:       "prefix",
			wantBypassAuth: true,
			description:    "Longest prefix should take priority",
		},
		{
			name: "longest_suffix",
			endpoints: []CORSEndpointConfig{
				{
					Path:        "*.json",
					PatternType: "suffix",
					BypassAuth:  false,
				},
				{
					Path:        "*.full.json",
					PatternType: "suffix",
					BypassAuth:  true,
				},
			},
			path:           "/reports/data.full.json",
			wantPath:       "*.full.json",
			wantType:       "suffix",
			wantBypassAuth: true,
			description:    "Longest suffix should take priority",
		},
		{
			name: "no_match",
			endpoints: []CORSEndpointConfig{
				{
					Path:        "/public/*",
					PatternType: "prefix",
					BypassAuth:  true,
				},
			},
			path:           "/private/data",
			wantPath:       "",
			wantType:       "",
			wantBypassAuth: false,
			description:    "No match should return nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewCORSMiddleware(CORSConfig{
				Endpoints: tt.endpoints,
			})

			got := m.MatchEndpoint(tt.path)

			if tt.wantPath == "" {
				if got != nil {
					t.Errorf("MatchEndpoint() = %v, want nil for %s", got, tt.description)
				}
				return
			}

			if got == nil {
				t.Errorf("MatchEndpoint() = nil, want match for %s", tt.description)
				return
			}

			if got.Path != tt.wantPath {
				t.Errorf("MatchEndpoint() path = %v, want %v for %s", got.Path, tt.wantPath, tt.description)
			}
			if got.PatternType != tt.wantType {
				t.Errorf("MatchEndpoint() type = %v, want %v for %s", got.PatternType, tt.wantType, tt.description)
			}
			if got.BypassAuth != tt.wantBypassAuth {
				t.Errorf("MatchEndpoint() bypassAuth = %v, want %v for %s", got.BypassAuth, tt.wantBypassAuth, tt.description)
			}
		})
	}
}

// TestCORSMiddleware_HandleDynamic tests dynamic CORS header application
func TestCORSMiddleware_HandleDynamic(t *testing.T) {
	tests := []struct {
		name             string
		endpoints        []CORSEndpointConfig
		path             string
		origin           string
		method           string
		wantStatus       int
		wantAllowOrigin  string
		wantAllowMethods string
		wantAllowHeaders string
		wantAllowCreds   string
		description      string
	}{
		{
			name: "wildcard_origin_public_endpoint",
			endpoints: []CORSEndpointConfig{
				{
					Path:             "/health",
					PatternType:      "exact",
					AllowedOrigins:   []string{"*"},
					AllowedMethods:   []string{"GET", "OPTIONS"},
					AllowedHeaders:   []string{"Content-Type"},
					AllowCredentials: false,
				},
			},
			path:             "/health",
			origin:           "https://example.com",
			method:           "GET",
			wantStatus:       200,
			wantAllowOrigin:  "*",
			wantAllowMethods: "GET, OPTIONS",
			wantAllowHeaders: "Content-Type",
			wantAllowCreds:   "",
			description:      "Wildcard origin should set Access-Control-Allow-Origin to *",
		},
		{
			name: "specific_origin_match",
			endpoints: []CORSEndpointConfig{
				{
					Path:             "/webhooks/*",
					PatternType:      "prefix",
					AllowedOrigins:   []string{"https://partner.example.com"},
					AllowedMethods:   []string{"POST", "OPTIONS"},
					AllowedHeaders:   []string{"Content-Type", "Authorization"},
					AllowCredentials: true,
				},
			},
			path:             "/webhooks/github",
			origin:           "https://partner.example.com",
			method:           "POST",
			wantStatus:       200,
			wantAllowOrigin:  "https://partner.example.com",
			wantAllowMethods: "POST, OPTIONS",
			wantAllowHeaders: "Content-Type, Authorization",
			wantAllowCreds:   "true",
			description:      "Specific origin match should set matching origin header",
		},
		{
			name: "specific_origin_no_match",
			endpoints: []CORSEndpointConfig{
				{
					Path:             "/webhooks/*",
					PatternType:      "prefix",
					AllowedOrigins:   []string{"https://partner.example.com"},
					AllowedMethods:   []string{"POST", "OPTIONS"},
					AllowedHeaders:   []string{"Content-Type"},
					AllowCredentials: false,
				},
			},
			path:             "/webhooks/github",
			origin:           "https://evil.com",
			method:           "POST",
			wantStatus:       200,
			wantAllowOrigin:  "", // No CORS header for non-allowed origin
			wantAllowMethods: "",
			wantAllowHeaders: "",
			wantAllowCreds:   "",
			description:      "Non-allowed origin should not set CORS headers",
		},
		{
			name: "options_preflight_returns_204",
			endpoints: []CORSEndpointConfig{
				{
					Path:             "/health",
					PatternType:      "exact",
					AllowedOrigins:   []string{"*"},
					AllowedMethods:   []string{"GET", "OPTIONS"},
					AllowedHeaders:   []string{"Content-Type"},
					AllowCredentials: false,
				},
			},
			path:             "/health",
			origin:           "https://example.com",
			method:           "OPTIONS",
			wantStatus:       204,
			wantAllowOrigin:  "*",
			wantAllowMethods: "GET, OPTIONS",
			wantAllowHeaders: "Content-Type",
			wantAllowCreds:   "",
			description:      "OPTIONS preflight should return 204 with CORS headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewCORSMiddleware(CORSConfig{
				MaxAge:    3600,
				Endpoints: tt.endpoints,
			})

			// Create test handler that always returns 200
			handler := m.HandleDynamic(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleDynamic() status = %v, want %v for %s", w.Code, tt.wantStatus, tt.description)
			}

			gotAllowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotAllowOrigin != tt.wantAllowOrigin {
				t.Errorf("HandleDynamic() Access-Control-Allow-Origin = %v, want %v for %s", gotAllowOrigin, tt.wantAllowOrigin, tt.description)
			}

			if tt.wantAllowMethods != "" {
				gotAllowMethods := w.Header().Get("Access-Control-Allow-Methods")
				if gotAllowMethods != tt.wantAllowMethods {
					t.Errorf("HandleDynamic() Access-Control-Allow-Methods = %v, want %v for %s", gotAllowMethods, tt.wantAllowMethods, tt.description)
				}
			}

			if tt.wantAllowHeaders != "" {
				gotAllowHeaders := w.Header().Get("Access-Control-Allow-Headers")
				if gotAllowHeaders != tt.wantAllowHeaders {
					t.Errorf("HandleDynamic() Access-Control-Allow-Headers = %v, want %v for %s", gotAllowHeaders, tt.wantAllowHeaders, tt.description)
				}
			}

			gotAllowCreds := w.Header().Get("Access-Control-Allow-Credentials")
			if gotAllowCreds != tt.wantAllowCreds {
				t.Errorf("HandleDynamic() Access-Control-Allow-Credentials = %v, want %v for %s", gotAllowCreds, tt.wantAllowCreds, tt.description)
			}
		})
	}
}

// TestCORSMiddleware_HandleDynamic_NoOrigin tests behavior when Origin header is missing
func TestCORSMiddleware_HandleDynamic_NoOrigin(t *testing.T) {
	m := NewCORSMiddleware(CORSConfig{
		Endpoints: []CORSEndpointConfig{
			{
				Path:           "/health",
				PatternType:    "exact",
				AllowedOrigins: []string{"*"},
			},
		},
	})

	handler := m.HandleDynamic(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/health", nil)
	// No Origin header set
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleDynamic() status = %v, want 200", w.Code)
	}

	// No CORS headers should be set when Origin is missing
	gotAllowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if gotAllowOrigin != "" {
		t.Errorf("HandleDynamic() Access-Control-Allow-Origin = %v, want empty when Origin header missing", gotAllowOrigin)
	}
}

// TestCORSMiddleware_HandleDynamic_FallbackToGlobal tests fallback to global CORS config
func TestCORSMiddleware_HandleDynamic_FallbackToGlobal(t *testing.T) {
	m := NewCORSMiddleware(CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://global.example.com"},
		AllowedMethods: []string{"GET", "POST"},
		MaxAge:         7200,
		Endpoints: []CORSEndpointConfig{
			{
				Path:           "/health",
				PatternType:    "exact",
				AllowedOrigins: []string{"*"},
			},
		},
	})

	handler := m.HandleDynamic(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Request to path not in registered endpoints
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Origin", "https://global.example.com")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleDynamic() status = %v, want 200", w.Code)
	}

	// Should use global CORS config
	gotAllowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if gotAllowOrigin != "https://global.example.com" {
		t.Errorf("HandleDynamic() Access-Control-Allow-Origin = %v, want https://global.example.com (global config)", gotAllowOrigin)
	}
}

// TestCORSMiddleware_HandleDynamic_EndpointOriginFallbackToGlobal tests endpoint with empty origins uses global origins
func TestCORSMiddleware_HandleDynamic_EndpointOriginFallbackToGlobal(t *testing.T) {
	tests := []struct {
		name             string
		globalOrigins    []string
		endpointOrigins  []string
		requestOrigin    string
		wantAllowOrigin  string
		wantAllowMethods string
		description      string
	}{
		{
			name:             "empty_endpoint_origins_uses_global",
			globalOrigins:    []string{"https://app.example.com", "https://admin.example.com"},
			endpointOrigins:  []string{}, // Empty, should fall back to global
			requestOrigin:    "https://app.example.com",
			wantAllowOrigin:  "https://app.example.com",
			wantAllowMethods: "POST, OPTIONS",
			description:      "Endpoint with empty origins should use global allowed origins",
		},
		{
			name:             "explicit_endpoint_origins_override_global",
			globalOrigins:    []string{"https://app.example.com"},
			endpointOrigins:  []string{"https://special.example.com"},
			requestOrigin:    "https://special.example.com",
			wantAllowOrigin:  "https://special.example.com",
			wantAllowMethods: "POST, OPTIONS",
			description:      "Endpoint with explicit origins should override global",
		},
		{
			name:             "wildcard_global_with_empty_endpoint",
			globalOrigins:    []string{"*"},
			endpointOrigins:  []string{}, // Empty, should fall back to global wildcard
			requestOrigin:    "https://any-origin.com",
			wantAllowOrigin:  "*",
			wantAllowMethods: "POST, OPTIONS",
			description:      "Endpoint with empty origins should inherit global wildcard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewCORSMiddleware(CORSConfig{
				Enabled:        true,
				AllowedOrigins: tt.globalOrigins,
				AllowedMethods: []string{"GET", "POST"},
				MaxAge:         3600,
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "*:create",
						PatternType:    "suffix",
						AllowedOrigins: tt.endpointOrigins,
						AllowedMethods: []string{"POST", "OPTIONS"},
						AllowedHeaders: []string{"Content-Type", "Authorization"},
					},
				},
			})

			handler := m.HandleDynamic(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("POST", "/users:create", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			w := httptest.NewRecorder()

			handler(w, req)

			gotAllowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotAllowOrigin != tt.wantAllowOrigin {
				t.Errorf("HandleDynamic() Access-Control-Allow-Origin = %v, want %v for %s",
					gotAllowOrigin, tt.wantAllowOrigin, tt.description)
			}

			if tt.wantAllowMethods != "" {
				gotAllowMethods := w.Header().Get("Access-Control-Allow-Methods")
				if gotAllowMethods != tt.wantAllowMethods {
					t.Errorf("HandleDynamic() Access-Control-Allow-Methods = %v, want %v for %s",
						gotAllowMethods, tt.wantAllowMethods, tt.description)
				}
			}
		})
	}
}

// TestCORSMiddleware_HandleDynamic_DataEndpoints tests CORS for dynamic data endpoints
func TestCORSMiddleware_HandleDynamic_DataEndpoints(t *testing.T) {
	tests := []struct {
		name             string
		endpoints        []CORSEndpointConfig
		path             string
		origin           string
		method           string
		wantStatus       int
		wantAllowOrigin  string
		wantAllowMethods string
		wantAllowHeaders string
		description      string
	}{
		{
			name: "list_endpoint_with_suffix_pattern",
			endpoints: []CORSEndpointConfig{
				{
					Path:           "*:list",
					PatternType:    "suffix",
					AllowedOrigins: []string{"https://app.example.com"},
					AllowedMethods: []string{"GET", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization"},
				},
			},
			path:             "/users:list",
			origin:           "https://app.example.com",
			method:           "GET",
			wantStatus:       200,
			wantAllowOrigin:  "https://app.example.com",
			wantAllowMethods: "GET, OPTIONS",
			wantAllowHeaders: "Content-Type, Authorization",
			description:      "GET /users:list should match *:list suffix pattern",
		},
		{
			name: "create_endpoint_with_suffix_pattern",
			endpoints: []CORSEndpointConfig{
				{
					Path:           "*:create",
					PatternType:    "suffix",
					AllowedOrigins: []string{"https://app.example.com"},
					AllowedMethods: []string{"POST", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization"},
				},
			},
			path:             "/users:create",
			origin:           "https://app.example.com",
			method:           "POST",
			wantStatus:       200,
			wantAllowOrigin:  "https://app.example.com",
			wantAllowMethods: "POST, OPTIONS",
			wantAllowHeaders: "Content-Type, Authorization",
			description:      "POST /users:create should match *:create suffix pattern",
		},
		{
			name: "create_endpoint_preflight_options",
			endpoints: []CORSEndpointConfig{
				{
					Path:           "*:create",
					PatternType:    "suffix",
					AllowedOrigins: []string{"https://app.example.com"},
					AllowedMethods: []string{"POST", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization"},
				},
			},
			path:             "/products:create",
			origin:           "https://app.example.com",
			method:           "OPTIONS",
			wantStatus:       204,
			wantAllowOrigin:  "https://app.example.com",
			wantAllowMethods: "POST, OPTIONS",
			wantAllowHeaders: "Content-Type, Authorization",
			description:      "OPTIONS /products:create should return 204 with CORS headers",
		},
		{
			name: "update_endpoint_with_suffix_pattern",
			endpoints: []CORSEndpointConfig{
				{
					Path:           "*:update",
					PatternType:    "suffix",
					AllowedOrigins: []string{"https://app.example.com"},
					AllowedMethods: []string{"POST", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization"},
				},
			},
			path:             "/orders:update",
			origin:           "https://app.example.com",
			method:           "POST",
			wantStatus:       200,
			wantAllowOrigin:  "https://app.example.com",
			wantAllowMethods: "POST, OPTIONS",
			wantAllowHeaders: "Content-Type, Authorization",
			description:      "POST /orders:update should match *:update suffix pattern",
		},
		{
			name: "destroy_endpoint_with_suffix_pattern",
			endpoints: []CORSEndpointConfig{
				{
					Path:           "*:destroy",
					PatternType:    "suffix",
					AllowedOrigins: []string{"https://app.example.com"},
					AllowedMethods: []string{"POST", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization"},
				},
			},
			path:             "/items:destroy",
			origin:           "https://app.example.com",
			method:           "POST",
			wantStatus:       200,
			wantAllowOrigin:  "https://app.example.com",
			wantAllowMethods: "POST, OPTIONS",
			wantAllowHeaders: "Content-Type, Authorization",
			description:      "POST /items:destroy should match *:destroy suffix pattern",
		},
		{
			name: "get_endpoint_with_suffix_pattern",
			endpoints: []CORSEndpointConfig{
				{
					Path:           "*:get",
					PatternType:    "suffix",
					AllowedOrigins: []string{"https://app.example.com"},
					AllowedMethods: []string{"GET", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization"},
				},
			},
			path:             "/customers:get",
			origin:           "https://app.example.com",
			method:           "GET",
			wantStatus:       200,
			wantAllowOrigin:  "https://app.example.com",
			wantAllowMethods: "GET, OPTIONS",
			wantAllowHeaders: "Content-Type, Authorization",
			description:      "GET /customers:get should match *:get suffix pattern",
		},
		{
			name: "schema_endpoint_with_suffix_pattern",
			endpoints: []CORSEndpointConfig{
				{
					Path:           "*:schema",
					PatternType:    "suffix",
					AllowedOrigins: []string{"https://app.example.com"},
					AllowedMethods: []string{"GET", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization"},
				},
			},
			path:             "/products:schema",
			origin:           "https://app.example.com",
			method:           "GET",
			wantStatus:       200,
			wantAllowOrigin:  "https://app.example.com",
			wantAllowMethods: "GET, OPTIONS",
			wantAllowHeaders: "Content-Type, Authorization",
			description:      "GET /products:schema should match *:schema suffix pattern",
		},
		{
			name: "wildcard_origin_for_data_endpoints",
			endpoints: []CORSEndpointConfig{
				{
					Path:           "*:list",
					PatternType:    "suffix",
					AllowedOrigins: []string{"*"},
					AllowedMethods: []string{"GET", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type"},
				},
			},
			path:             "/any-collection:list",
			origin:           "https://any-origin.com",
			method:           "GET",
			wantStatus:       200,
			wantAllowOrigin:  "*",
			wantAllowMethods: "GET, OPTIONS",
			wantAllowHeaders: "Content-Type",
			description:      "Wildcard origin should work for data endpoints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewCORSMiddleware(CORSConfig{
				MaxAge:    3600,
				Endpoints: tt.endpoints,
			})

			handler := m.HandleDynamic(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleDynamic() status = %v, want %v for %s", w.Code, tt.wantStatus, tt.description)
			}

			gotAllowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotAllowOrigin != tt.wantAllowOrigin {
				t.Errorf("HandleDynamic() Access-Control-Allow-Origin = %v, want %v for %s", gotAllowOrigin, tt.wantAllowOrigin, tt.description)
			}

			if tt.wantAllowMethods != "" {
				gotAllowMethods := w.Header().Get("Access-Control-Allow-Methods")
				if gotAllowMethods != tt.wantAllowMethods {
					t.Errorf("HandleDynamic() Access-Control-Allow-Methods = %v, want %v for %s", gotAllowMethods, tt.wantAllowMethods, tt.description)
				}
			}

			if tt.wantAllowHeaders != "" {
				gotAllowHeaders := w.Header().Get("Access-Control-Allow-Headers")
				if gotAllowHeaders != tt.wantAllowHeaders {
					t.Errorf("HandleDynamic() Access-Control-Allow-Headers = %v, want %v for %s", gotAllowHeaders, tt.wantAllowHeaders, tt.description)
				}
			}
		})
	}
}
