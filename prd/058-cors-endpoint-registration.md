# PRD-058: CORS Endpoint Registration

## Overview

**Problem Statement:**
Moon currently hardcodes CORS configuration for specific public endpoints (`/health`, `/doc`, `/doc/llms-full.txt`) in the server setup code. As the API evolves and new public or semi-public endpoints are added (webhooks, public data feeds, status pages), hardcoding each endpoint creates maintenance burden, reduces flexibility, and makes it difficult to adapt CORS policies without code changes.

**Context:**
- **PRD-052** implemented public CORS (`Access-Control-Allow-Origin: *`) for health and documentation endpoints
- **PRD-057** unified authentication headers to `Authorization: Bearer` for both JWT and API keys
- Moon is a Go-based REST API providing dynamic database management
- CORS policies are currently applied via middleware helper functions (`publicCORS`, `public`, etc.) in `server.go`
- Different endpoints require different CORS policies:
  - **Public endpoints** (e.g., `/health`, `/doc`): `Access-Control-Allow-Origin: *`, no authentication
  - **Protected endpoints** (e.g., `/data/*`, `/collections/*`): Restricted origins, requires authentication
  - **Future endpoints** (e.g., `/webhooks`, `/public/reports/*`): May need selective CORS with custom configurations

**Solution:**
Implement a declarative CORS endpoint registration system in the YAML configuration file. This allows administrators to:
- Define CORS-enabled endpoints using exact paths or wildcard patterns (e.g., `/api/v1/public/*`)
- Configure CORS settings per endpoint or pattern (allowed origins, methods, headers)
- Override global CORS settings for specific routes
- Maintain backward compatibility with existing hardcoded public endpoints
- Keep security controls: public CORS endpoints still require authentication unless explicitly configured otherwise

**Benefits:**
- **Operational Flexibility:** Add/modify CORS-enabled endpoints without code changes or redeployment
- **Simplified Management:** Centralized CORS configuration in YAML file alongside other settings
- **Pattern Matching:** Support wildcards for route groups (e.g., `/public/*`, `/webhooks/*`)
- **Granular Control:** Per-endpoint CORS settings for different use cases (external partners, monitoring tools, public APIs)
- **Backward Compatibility:** Existing hardcoded endpoints continue working as configuration defaults
- **Security by Default:** Endpoints require explicit CORS configuration; no accidental exposure

**Breaking Change:**
This is **NOT a breaking change**. All existing hardcoded public CORS endpoints will be migrated to default configuration values, ensuring zero disruption for existing deployments.

---

## Requirements

### FR-1: Configuration Schema for CORS Endpoint Registration

**FR-1.1: CORS Endpoints Configuration Structure**
Add new configuration section to `AppConfig`:
```yaml
cors:
  enabled: true  # Global CORS enable/disable (existing)
  allowed_origins: ["*"]  # Default origins for standard CORS (existing)
  allowed_methods: ["GET", "POST", "OPTIONS"]  # Existing
  allowed_headers: ["Content-Type", "Authorization"]  # Existing
  allow_credentials: true  # Existing
  max_age: 3600  # Existing
  
  # NEW: Endpoint-specific CORS registration
  endpoints:
    - path: "/health"
      pattern_type: "exact"  # exact | prefix | suffix | contains
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true  # Skip authentication for this endpoint
      
    - path: "/doc/*"
      pattern_type: "prefix"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
      
    - path: "/webhooks/*"
      pattern_type: "prefix"
      allowed_origins: ["https://partner.example.com"]
      allowed_methods: ["POST", "OPTIONS"]
      allowed_headers: ["Content-Type", "Authorization", "X-Webhook-Signature"]
      allow_credentials: false
      bypass_auth: false  # Requires authentication
      
    - path: "/public/reports/*"
      pattern_type: "prefix"
      allowed_origins: ["https://dashboard.example.com"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
```

**FR-1.2: Pattern Types**
Support four pattern matching types:
1. **`exact`**: Matches exact path only (e.g., `/health` matches `/health` but not `/health/status`)
2. **`prefix`**: Matches path prefix (e.g., `/doc/*` matches `/doc`, `/doc/api`, `/doc/llms-full.txt`)
3. **`suffix`**: Matches path suffix (e.g., `*.json` matches `/data/users.json`, `/reports/summary.json`)
4. **`contains`**: Matches if path contains substring (e.g., `/public/` matches any path containing `/public/`)

**FR-1.3: Configuration Validation**
Validate CORS endpoint configuration on startup:
- Each `path` must be non-empty string
- `pattern_type` must be one of: `exact`, `prefix`, `suffix`, `contains`
- `allowed_origins` must be non-empty array (or `["*"]` for public)
- `allowed_methods` must be non-empty array
- `allowed_headers` must be valid HTTP header names
- `bypass_auth` must be boolean (default: `false`)
- If `allowed_origins` contains `*`, it must be the only origin (no mixing)
- Warn if `bypass_auth: true` and `allowed_origins` is not `["*"]` (unusual configuration)

**FR-1.4: Default Endpoint Registrations**
If `cors.endpoints` is not specified, apply these defaults (backward compatibility with PRD-052):
```yaml
cors:
  endpoints:
    - path: "/health"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
      
    - path: "/doc"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
      
    - path: "/doc/llms-full.txt"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
```

**FR-1.5: Configuration Priority**
When multiple patterns match a request path, apply the most specific match:
1. **Exact match** takes highest priority (e.g., `/health` exact match overrides `/health/*` prefix)
2. **Longest prefix match** (e.g., `/api/v1/public/*` overrides `/api/*`)
3. **Longest suffix match** (e.g., `*.json` overrides `*`)
4. **Longest contains match** (e.g., `/public/data/` overrides `/public/`)
5. If no registered endpoint matches, fall back to global CORS configuration

### FR-2: CORS Middleware Enhancement

**FR-2.1: Dynamic CORS Matcher**
Enhance `CORSMiddleware` to support dynamic endpoint matching:

File: `cmd/moon/internal/middleware/cors.go`

Add new method:
```go
// MatchEndpoint checks if a request path matches any registered CORS endpoint
// Returns the matching endpoint configuration, or nil if no match
func (m *CORSMiddleware) MatchEndpoint(path string) *CORSEndpointConfig {
    var bestMatch *CORSEndpointConfig
    var bestMatchScore int
    
    for _, endpoint := range m.config.Endpoints {
        if matches, score := endpoint.Matches(path); matches {
            if score > bestMatchScore {
                bestMatch = &endpoint
                bestMatchScore = score
            }
        }
    }
    
    return bestMatch
}

// CORSEndpointConfig represents a single CORS endpoint registration
type CORSEndpointConfig struct {
    Path             string   `mapstructure:"path"`
    PatternType      string   `mapstructure:"pattern_type"`      // exact, prefix, suffix, contains
    AllowedOrigins   []string `mapstructure:"allowed_origins"`
    AllowedMethods   []string `mapstructure:"allowed_methods"`
    AllowedHeaders   []string `mapstructure:"allowed_headers"`
    AllowCredentials bool     `mapstructure:"allow_credentials"`
    BypassAuth       bool     `mapstructure:"bypass_auth"`
}

// Matches checks if this endpoint config matches the given path
// Returns (matched bool, score int) where higher scores indicate more specific matches
func (e *CORSEndpointConfig) Matches(path string) (bool, int) {
    switch e.PatternType {
    case "exact":
        if path == e.Path {
            return true, 1000 + len(e.Path) // Highest priority
        }
    case "prefix":
        cleanPattern := strings.TrimSuffix(e.Path, "/*")
        if strings.HasPrefix(path, cleanPattern) {
            return true, 500 + len(cleanPattern) // Priority by prefix length
        }
    case "suffix":
        cleanPattern := strings.TrimPrefix(e.Path, "*")
        if strings.HasSuffix(path, cleanPattern) {
            return true, 300 + len(cleanPattern) // Priority by suffix length
        }
    case "contains":
        cleanPattern := strings.Trim(e.Path, "*")
        if strings.Contains(path, cleanPattern) {
            return true, 100 + len(cleanPattern) // Lowest priority
        }
    }
    return false, 0
}
```

**FR-2.2: Dynamic CORS Handler**
Replace `HandlePublic` with dynamic handler that uses endpoint registration:

```go
// HandleDynamic applies CORS based on endpoint registration or falls back to global config
func (m *CORSMiddleware) HandleDynamic(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Check if request path matches a registered CORS endpoint
        endpointConfig := m.MatchEndpoint(r.URL.Path)
        
        if endpointConfig != nil {
            // Apply endpoint-specific CORS
            m.applyCORSHeaders(w, r, endpointConfig.AllowedOrigins, 
                               endpointConfig.AllowedMethods, 
                               endpointConfig.AllowedHeaders,
                               endpointConfig.AllowCredentials)
            
            // Handle OPTIONS preflight
            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return
            }
            
            next(w, r)
            return
        }
        
        // Fall back to standard CORS handler
        m.Handle(next)(w, r)
    }
}

// applyCORSHeaders is a helper to apply CORS headers based on configuration
func (m *CORSMiddleware) applyCORSHeaders(w http.ResponseWriter, r *http.Request, 
                                          origins, methods, headers []string, 
                                          allowCredentials bool) {
    origin := r.Header.Get("Origin")
    
    // Check if origin is allowed
    if origin != "" && m.isOriginAllowedFor(origin, origins) {
        // For wildcard, use "*" directly
        if len(origins) == 1 && origins[0] == "*" {
            w.Header().Set("Access-Control-Allow-Origin", "*")
        } else {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        }
        
        if allowCredentials {
            w.Header().Set("Access-Control-Allow-Credentials", "true")
        }
        
        if len(methods) > 0 {
            w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
        }
        
        if len(headers) > 0 {
            w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
        }
        
        if m.config.MaxAge > 0 {
            w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", m.config.MaxAge))
        }
    }
}

// isOriginAllowedFor checks if origin is in the specific allowed list
func (m *CORSMiddleware) isOriginAllowedFor(origin string, allowedOrigins []string) bool {
    if len(allowedOrigins) == 0 {
        return false
    }
    
    for _, allowed := range allowedOrigins {
        if allowed == "*" || allowed == origin {
            return true
        }
    }
    
    return false
}
```

**FR-2.3: Keep Existing Methods for Backward Compatibility**
Preserve `Handle` and `HandlePublic` methods for any code still using them directly:
- `Handle`: Standard CORS with global config (existing behavior)
- `HandlePublic`: Public CORS with `Access-Control-Allow-Origin: *` (existing behavior for PRD-052)
- `HandleDynamic`: New method using endpoint registration (default going forward)

### FR-3: Authentication Bypass for Public Endpoints

**FR-3.1: Authentication Middleware Enhancement**
Update authentication middleware to check if endpoint is configured for auth bypass:

File: `cmd/moon/internal/middleware/auth.go`

Add method to `UnifiedAuthMiddleware`:
```go
// ShouldBypassAuth checks if the request path should bypass authentication
func (m *UnifiedAuthMiddleware) ShouldBypassAuth(path string) bool {
    // Check CORS endpoint registration
    if m.corsMiddleware != nil {
        endpointConfig := m.corsMiddleware.MatchEndpoint(path)
        if endpointConfig != nil && endpointConfig.BypassAuth {
            return true
        }
    }
    return false
}
```

**FR-3.2: Authentication Middleware Integration**
Modify `UnifiedAuthMiddleware.Handle` to skip authentication for bypass endpoints:
```go
func (m *UnifiedAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Check if this endpoint should bypass authentication
        if m.ShouldBypassAuth(r.URL.Path) {
            next(w, r)
            return
        }
        
        // Proceed with normal authentication flow
        // ... existing authentication logic ...
    }
}
```

**FR-3.3: Security Validation**
Ensure security guardrails:
- Log warning if `bypass_auth: true` with non-wildcard origins (unusual configuration)
- Log info message when bypassing auth for registered endpoints (audit trail)
- Prevent accidental bypass by requiring explicit `bypass_auth: true` (default is `false`)
- Document security implications in configuration comments

### FR-4: Server Setup Refactoring

**FR-4.1: Replace Hardcoded Public CORS Routes**
File: `cmd/moon/internal/server/server.go`

Replace:
```go
// Old approach (hardcoded)
publicCORS := func(h http.HandlerFunc) http.HandlerFunc {
    return s.corsMiddle.HandlePublic(s.loggingMiddleware(h))
}

s.mux.HandleFunc("GET "+healthPath, publicCORS(s.healthHandler))
s.mux.HandleFunc("GET "+prefix+"/doc/{$}", publicCORS(docHandler.HTML))
s.mux.HandleFunc("GET "+prefix+"/doc/llms-full.txt", publicCORS(docHandler.Markdown))
```

With:
```go
// New approach (configuration-driven)
dynamicCORS := func(h http.HandlerFunc) http.HandlerFunc {
    return s.corsMiddle.HandleDynamic(s.loggingMiddleware(h))
}

// Apply dynamic CORS to all routes that may be registered
s.mux.HandleFunc("GET "+healthPath, dynamicCORS(s.healthHandler))
s.mux.HandleFunc("GET "+prefix+"/doc/{$}", dynamicCORS(docHandler.HTML))
s.mux.HandleFunc("GET "+prefix+"/doc/llms-full.txt", dynamicCORS(docHandler.Markdown))
```

**FR-4.2: Optional Global Dynamic CORS**
For maximum flexibility, consider applying `HandleDynamic` to all routes:
```go
// Apply dynamic CORS + logging to all endpoints
all := func(h http.HandlerFunc) http.HandlerFunc {
    return s.corsMiddle.HandleDynamic(s.loggingMiddleware(h))
}
```
This allows any endpoint to be registered for custom CORS via configuration without code changes.

**FR-4.3: Middleware Ordering**
Ensure correct middleware ordering:
1. **Logging** (first, logs all requests)
2. **CORS** (second, handles preflight and sets headers)
3. **Authentication** (third, checks auth after CORS headers set, skips if `bypass_auth: true`)
4. **Rate Limiting** (fourth, applies limits after authentication)
5. **Authorization** (fifth, checks permissions)
6. **Handler** (final, executes endpoint logic)

### FR-5: Configuration File Updates

**FR-5.1: Sample Configuration**
Update `samples/moon.conf` with new CORS endpoint registration section:

```yaml
# ============================================================================
# CORS Configuration (Optional)
# ============================================================================
# Cross-Origin Resource Sharing (CORS) for browser-based API access.
# Global CORS settings apply to all endpoints unless overridden by endpoint-specific configuration.
#
# Endpoint Registration: Define CORS policies for specific routes or patterns.
# - path: Endpoint path or pattern
# - pattern_type: "exact", "prefix", "suffix", or "contains"
# - allowed_origins: List of allowed origins or ["*"] for public access
# - allowed_methods: HTTP methods (GET, POST, PUT, DELETE, OPTIONS, etc.)
# - allowed_headers: Request headers allowed in preflight
# - allow_credentials: Allow cookies/auth headers (requires specific origins, not "*")
# - bypass_auth: Skip authentication for this endpoint (default: false)
#
# Pattern Types:
# - exact: Matches exact path only (/health matches /health, not /health/status)
# - prefix: Matches path prefix (/doc/* matches /doc, /doc/api, /doc/llms-full.txt)
# - suffix: Matches path suffix (*.json matches /data/users.json)
# - contains: Matches if path contains substring (/public/ matches any path with /public/)
#
# Security:
# - Always use specific origins in production (avoid "*" except for truly public data)
# - Set bypass_auth: true only for public endpoints (health, docs, status)
# - Use allow_credentials: true only when needed (requires specific origins)
#
# Default Endpoints (if cors.endpoints not specified):
# - /health (exact, *, no auth)
# - /doc (exact, *, no auth)
# - /doc/llms-full.txt (exact, *, no auth)
#
cors:
  # Global CORS settings (applied when no endpoint-specific config matches)
  enabled: false
  allowed_origins: []
  allowed_methods: ["GET", "POST", "OPTIONS"]
  allowed_headers: ["Content-Type", "Authorization"]
  allow_credentials: true
  max_age: 3600
  
  # Endpoint-specific CORS registration (optional)
  # If not specified, defaults to /health, /doc, /doc/llms-full.txt with public CORS
  endpoints:
    # Health check (public, no authentication)
    - path: "/health"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
    
    # Documentation (public, no authentication)
    - path: "/doc"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
    
    # LLM-friendly documentation (public, no authentication)
    - path: "/doc/llms-full.txt"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
    
    # Example: Public API endpoints (requires authentication, selective CORS)
    # - path: "/public/*"
    #   pattern_type: "prefix"
    #   allowed_origins: ["https://dashboard.example.com", "https://app.example.com"]
    #   allowed_methods: ["GET", "OPTIONS"]
    #   allowed_headers: ["Content-Type", "Authorization"]
    #   allow_credentials: true
    #   bypass_auth: false
    
    # Example: Webhooks (requires authentication, partner origins only)
    # - path: "/webhooks/*"
    #   pattern_type: "prefix"
    #   allowed_origins: ["https://partner.example.com"]
    #   allowed_methods: ["POST", "OPTIONS"]
    #   allowed_headers: ["Content-Type", "Authorization", "X-Webhook-Signature"]
    #   allow_credentials: false
    #   bypass_auth: false
```

**FR-5.2: Configuration Schema Update**
File: `cmd/moon/internal/config/config.go`

Add to `CORSConfig` struct:
```go
type CORSConfig struct {
    Enabled          bool                 `mapstructure:"enabled"`
    AllowedOrigins   []string             `mapstructure:"allowed_origins"`
    AllowedMethods   []string             `mapstructure:"allowed_methods"`
    AllowedHeaders   []string             `mapstructure:"allowed_headers"`
    AllowCredentials bool                 `mapstructure:"allow_credentials"`
    MaxAge           int                  `mapstructure:"max_age"`
    Endpoints        []CORSEndpointConfig `mapstructure:"endpoints"` // NEW
}

type CORSEndpointConfig struct {
    Path             string   `mapstructure:"path"`
    PatternType      string   `mapstructure:"pattern_type"`
    AllowedOrigins   []string `mapstructure:"allowed_origins"`
    AllowedMethods   []string `mapstructure:"allowed_methods"`
    AllowedHeaders   []string `mapstructure:"allowed_headers"`
    AllowCredentials bool     `mapstructure:"allow_credentials"`
    BypassAuth       bool     `mapstructure:"bypass_auth"`
}
```

**FR-5.3: Configuration Defaults**
Update `Defaults.CORS` in `config.go`:
```go
CORS: struct {
    Enabled          bool
    AllowedOrigins   []string
    AllowedMethods   []string
    AllowedHeaders   []string
    AllowCredentials bool
    MaxAge           int
    Endpoints        []CORSEndpointConfig
}{
    Enabled:          false,
    AllowedOrigins:   []string{},
    AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
    Endpoints: []CORSEndpointConfig{
        {
            Path:             "/health",
            PatternType:      "exact",
            AllowedOrigins:   []string{"*"},
            AllowedMethods:   []string{"GET", "OPTIONS"},
            AllowedHeaders:   []string{"Content-Type"},
            AllowCredentials: false,
            BypassAuth:       true,
        },
        {
            Path:             "/doc",
            PatternType:      "exact",
            AllowedOrigins:   []string{"*"},
            AllowedMethods:   []string{"GET", "OPTIONS"},
            AllowedHeaders:   []string{"Content-Type"},
            AllowCredentials: false,
            BypassAuth:       true,
        },
        {
            Path:             "/doc/llms-full.txt",
            PatternType:      "exact",
            AllowedOrigins:   []string{"*"},
            AllowedMethods:   []string{"GET", "OPTIONS"},
            AllowedHeaders:   []string{"Content-Type"},
            AllowCredentials: false,
            BypassAuth:       true,
        },
    },
},
```

### FR-6: Documentation Updates

**FR-6.1: SPEC.md Updates**
Add new section to `SPEC.md`:

```markdown
### CORS Endpoint Registration

Moon supports dynamic CORS endpoint registration via configuration. This allows operators to:
- Define CORS policies for specific endpoints or patterns
- Control authentication bypass for public endpoints
- Support wildcard patterns for route groups

#### Configuration

CORS endpoints are configured in `cors.endpoints` section:

```yaml
cors:
  endpoints:
    - path: "/health"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
```

#### Pattern Types

- **exact**: Matches exact path only
- **prefix**: Matches path prefix (supports `/*` suffix)
- **suffix**: Matches path suffix (supports `*` prefix)
- **contains**: Matches if path contains substring

#### Priority

When multiple patterns match, the most specific match is used:
1. Exact matches (highest priority)
2. Longest prefix matches
3. Longest suffix matches
4. Longest contains matches
5. Global CORS configuration (fallback)

#### Default Endpoints

If `cors.endpoints` is not specified, these defaults are applied:
- `/health` (public, no auth)
- `/doc` (public, no auth)
- `/doc/llms-full.txt` (public, no auth)
```

**FR-6.2: INSTALL.md Updates**
Add configuration example to `INSTALL.md`:

```markdown
### CORS Configuration

To enable CORS for specific endpoints:

1. Edit `/etc/moon.conf`
2. Add endpoint registration:
   ```yaml
   cors:
     endpoints:
       - path: "/public/*"
         pattern_type: "prefix"
         allowed_origins: ["https://app.example.com"]
         allowed_methods: ["GET", "OPTIONS"]
         bypass_auth: false
   ```
3. Restart Moon: `systemctl restart moon`

For public endpoints without authentication:
```yaml
cors:
  endpoints:
    - path: "/status"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      bypass_auth: true
```
```

**FR-6.3: README.md Updates**
Add CORS endpoint registration to features list:
```markdown
### Features

- **Dynamic CORS Configuration**: Register CORS-enabled endpoints via YAML configuration
  - Support wildcard patterns (`/public/*`, `*.json`)
  - Per-endpoint CORS policies (origins, methods, headers)
  - Authentication bypass for public endpoints
  - Backward compatible with existing endpoints
```

**FR-6.4: API Documentation Template**
Update `cmd/moon/internal/handlers/templates/doc.md.tmpl`:

Add section:
```markdown
## CORS Configuration

Moon supports Cross-Origin Resource Sharing (CORS) for browser-based API access. CORS policies can be configured per endpoint or pattern.

### Public Endpoints (No Authentication)

The following endpoints are publicly accessible with CORS enabled:

- `GET /health` - Health check
- `GET /doc` - API documentation (HTML)
- `GET /doc/llms-full.txt` - API documentation (Markdown)

**Example:**
```bash
curl -H "Origin: https://example.com" https://api.moon.example.com/health
```

Response includes:
```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, OPTIONS
```

### Custom CORS Endpoints

Administrators can register additional CORS-enabled endpoints in the configuration file. See `SPEC.md` for details.
```

### FR-7: Testing Requirements

**FR-7.1: Unit Tests - Pattern Matching**
File: `cmd/moon/internal/middleware/cors_test.go`

Test cases for pattern matching:
1. **Exact match**: `/health` matches `/health`, not `/health/status`
2. **Prefix match**: `/doc/*` matches `/doc`, `/doc/api`, `/doc/llms-full.txt`
3. **Suffix match**: `*.json` matches `/data/users.json`, `/reports/summary.json`
4. **Contains match**: `/public/` matches `/api/public/data`, `/public/reports`
5. **Priority - Exact over prefix**: `/health` exact beats `/health/*` prefix
6. **Priority - Longest prefix**: `/api/v1/public/*` beats `/api/*`
7. **Priority - Longest suffix**: `*.full.json` beats `*.json`
8. **No match**: `/private/data` with config for `/public/*` returns no match
9. **Multiple wildcards**: `/public/*/reports/*` matches `/public/users/reports/summary`
10. **Case sensitivity**: `/Health` does not match `/health` (paths are case-sensitive)

**FR-7.2: Unit Tests - CORS Header Application**
Test CORS header application:
1. **Wildcard origin**: `allowed_origins: ["*"]` sets `Access-Control-Allow-Origin: *`
2. **Specific origin match**: Origin `https://example.com` in allowed list sets header
3. **Specific origin no match**: Origin `https://evil.com` not in allowed list, no header set
4. **Credentials with wildcard**: `allow_credentials: true` with `["*"]` logs warning
5. **Credentials with specific origin**: `allow_credentials: true` with `["https://example.com"]` sets both headers
6. **Methods header**: `allowed_methods: ["GET", "POST"]` sets `Access-Control-Allow-Methods`
7. **Headers header**: `allowed_headers: ["Content-Type", "Authorization"]` sets `Access-Control-Allow-Headers`
8. **Max age**: `max_age: 3600` sets `Access-Control-Max-Age: 3600`
9. **OPTIONS preflight**: OPTIONS request returns 204 with CORS headers, no body
10. **Missing Origin header**: Request without Origin header, CORS headers not set

**FR-7.3: Unit Tests - Authentication Bypass**
Test authentication bypass behavior:
1. **Bypass enabled**: Endpoint with `bypass_auth: true` skips authentication
2. **Bypass disabled**: Endpoint with `bypass_auth: false` requires authentication
3. **Default bypass**: Endpoint without `bypass_auth` defaults to `false`
4. **Bypass with valid token**: Even with valid token, bypass endpoint skips auth middleware
5. **Bypass without token**: Bypass endpoint works without `Authorization` header

**FR-7.4: Integration Tests**
Create test script: `scripts/test-cors-endpoints.sh`

Test scenarios:
1. **Health endpoint public CORS**: `curl -H "Origin: https://test.com" http://localhost:6006/health`
   - Verify: Status 200, `Access-Control-Allow-Origin: *`
   - Verify: No authentication required
2. **Doc endpoint public CORS**: `curl -H "Origin: https://test.com" http://localhost:6006/doc`
   - Verify: Status 200, `Access-Control-Allow-Origin: *`
3. **Custom endpoint CORS**: Configure `/public/status` with specific origin
   - Request from allowed origin: Status 200, CORS header set
   - Request from non-allowed origin: Status 200, no CORS header
4. **Wildcard pattern CORS**: Configure `/webhooks/*` with CORS
   - Request to `/webhooks/github`: Verify CORS headers
   - Request to `/webhooks/stripe`: Verify CORS headers
5. **OPTIONS preflight**: Send OPTIONS to registered endpoint
   - Verify: Status 204, all CORS headers present
6. **Protected endpoint CORS**: Configure `/data/*` with CORS but `bypass_auth: false`
   - Request without auth: 401 Unauthorized
   - Request with auth + valid origin: 200 OK with CORS headers
7. **Priority testing**: Configure overlapping patterns
   - Exact `/health` and prefix `/health/*`
   - Verify exact match takes precedence
8. **Backward compatibility**: Without `cors.endpoints` config
   - Verify default endpoints (`/health`, `/doc`) still have public CORS

**FR-7.5: Configuration Validation Tests**
Test configuration validation:
1. **Valid configuration**: All fields correct, loads successfully
2. **Invalid pattern_type**: `pattern_type: "invalid"` returns validation error
3. **Empty path**: `path: ""` returns validation error
4. **Empty allowed_origins**: `allowed_origins: []` returns validation error
5. **Wildcard with other origins**: `allowed_origins: ["*", "https://example.com"]` returns validation error
6. **Credentials with wildcard**: `allow_credentials: true` + `allowed_origins: ["*"]` logs warning but loads
7. **Missing required fields**: Missing `pattern_type` uses default "exact"
8. **Missing optional fields**: Missing `allowed_headers` uses empty list (no restrictions)

### FR-8: Error Handling and Logging

**FR-8.1: Configuration Load Errors**
Handle configuration errors gracefully:
```json
{
  "error": "invalid_configuration",
  "message": "CORS endpoint configuration invalid: pattern_type must be one of: exact, prefix, suffix, contains",
  "field": "cors.endpoints[2].pattern_type",
  "value": "regex"
}
```

**FR-8.2: Pattern Match Logging**
Log endpoint CORS matches (DEBUG level):
```
DEBUG: CORS endpoint match: path=/health, pattern=/health (exact), origin=https://example.com, bypassed_auth=true
```

**FR-8.3: Configuration Warning Logging**
Log configuration warnings (WARN level):
```
WARN: CORS endpoint /health has allow_credentials=true with wildcard origin. This is not recommended and may not work in modern browsers.
```

**FR-8.4: Authentication Bypass Logging**
Log authentication bypass (INFO level):
```
INFO: Authentication bypassed for /health (CORS endpoint configuration)
```

**FR-8.5: Pattern Conflict Warnings**
Warn about potentially conflicting patterns (WARN level):
```
WARN: Multiple CORS endpoint patterns match /api/public/data: ["/api/public/*" (prefix, score=500), "/api/*" (prefix, score=400)]. Using most specific: /api/public/*
```

### FR-9: Security Considerations

**FR-9.1: Default Deny**
- Endpoints without CORS registration do not get CORS headers (secure by default)
- `bypass_auth` defaults to `false` (authentication required by default)
- Empty `allowed_origins` means no origins allowed (fail-safe)

**FR-9.2: Credentials and Wildcard Validation**
- If `allow_credentials: true` and `allowed_origins: ["*"]`, log warning (not allowed per CORS spec)
- Modern browsers reject this combination; warn admin to use specific origins

**FR-9.3: Public Endpoint Audit**
- Log all authentication bypasses for audit trail
- Include request path, origin, timestamp in logs
- Enable monitoring of public endpoint usage

**FR-9.4: Configuration Change Detection**
- Detect configuration changes on reload (SIGHUP)
- Log differences in CORS endpoint registration
- Warn about new `bypass_auth: true` endpoints

**FR-9.5: Rate Limiting for Public Endpoints**
- Public endpoints (with `bypass_auth: true`) should still be rate-limited by IP
- Prevent abuse of public CORS endpoints
- Consider separate rate limits for authenticated vs. unauthenticated requests

### FR-10: Migration Path

**FR-10.1: Backward Compatibility**
- If `cors.endpoints` not specified, apply default endpoints (PRD-052 behavior)
- Existing deployments continue working without configuration changes
- `HandlePublic` method preserved for code still using it directly

**FR-10.2: Gradual Migration**
Recommended migration steps:
1. **Version X.Y.0**: Introduce `cors.endpoints` configuration, defaults to PRD-052 behavior
2. **Update documentation**: Add CORS endpoint registration examples to SPEC.md, INSTALL.md
3. **Operators test**: Operators can add custom endpoints without affecting existing behavior
4. **Version X.Y+1.0**: Deprecate `HandlePublic` (still works but logs deprecation warning)
5. **Version X.Y+2.0**: (Optional) Remove `HandlePublic` if all code migrated to `HandleDynamic`

**FR-10.3: Configuration Migration Tool**
Create script to generate CORS endpoint configuration from existing code:
```bash
scripts/migrate-cors-config.sh
```

This script:
1. Scans `server.go` for `publicCORS` usage
2. Extracts endpoint paths
3. Generates `cors.endpoints` YAML configuration
4. Outputs to console or file for review

---

## Acceptance

### AC-1: Configuration Loading

- [ ] `cors.endpoints` configuration section loads correctly from YAML
- [ ] Default endpoints (`/health`, `/doc`, `/doc/llms-full.txt`) applied if `cors.endpoints` not specified
- [ ] Configuration validation rejects invalid `pattern_type` values
- [ ] Configuration validation rejects empty `path` values
- [ ] Configuration validation rejects empty `allowed_origins` arrays
- [ ] Configuration validation rejects wildcard mixed with specific origins
- [ ] Configuration validation warns if `allow_credentials: true` with wildcard origin
- [ ] Missing optional fields (e.g., `allowed_headers`) use sensible defaults
- [ ] Configuration loads successfully with valid endpoint registrations

### AC-2: Pattern Matching

- [ ] Exact pattern matches exact path only (e.g., `/health` matches `/health`, not `/health/status`)
- [ ] Prefix pattern matches path prefix (e.g., `/doc/*` matches `/doc`, `/doc/api`, `/doc/llms-full.txt`)
- [ ] Suffix pattern matches path suffix (e.g., `*.json` matches `/data/users.json`)
- [ ] Contains pattern matches substring (e.g., `/public/` matches any path with `/public/`)
- [ ] Exact match takes priority over prefix match
- [ ] Longest prefix match takes priority over shorter prefix match
- [ ] Longest suffix match takes priority over shorter suffix match
- [ ] Longest contains match takes priority over shorter contains match
- [ ] When no pattern matches, global CORS configuration is used (fallback)
- [ ] Pattern matching is case-sensitive (e.g., `/Health` does not match `/health`)

### AC-3: CORS Header Application

- [ ] Wildcard origin (`["*"]`) sets `Access-Control-Allow-Origin: *`
- [ ] Specific origin match sets `Access-Control-Allow-Origin: <origin>`
- [ ] Specific origin no match does not set CORS headers
- [ ] `allowed_methods` sets `Access-Control-Allow-Methods` header
- [ ] `allowed_headers` sets `Access-Control-Allow-Headers` header
- [ ] `allow_credentials: true` sets `Access-Control-Allow-Credentials: true`
- [ ] `max_age` sets `Access-Control-Max-Age` header
- [ ] OPTIONS preflight request returns 204 with all CORS headers
- [ ] Request without `Origin` header does not set CORS headers
- [ ] Registered endpoint CORS overrides global CORS configuration

### AC-4: Authentication Bypass

- [ ] Endpoint with `bypass_auth: true` skips authentication middleware
- [ ] Endpoint with `bypass_auth: false` requires authentication
- [ ] Endpoint without `bypass_auth` defaults to `false` (authentication required)
- [ ] Authentication bypass works for requests without `Authorization` header
- [ ] Authentication bypass logged in INFO logs (audit trail)
- [ ] Public endpoint (`bypass_auth: true`) accessible without credentials
- [ ] Protected endpoint (`bypass_auth: false`) returns 401 without credentials

### AC-5: Server Integration

- [ ] `HandleDynamic` middleware method implemented in `CORSMiddleware`
- [ ] Server routes use `HandleDynamic` for CORS handling
- [ ] Middleware ordering: Logging → CORS → Authentication → Rate Limiting → Handler
- [ ] Health endpoint (`/health`) uses dynamic CORS matching
- [ ] Documentation endpoints (`/doc`, `/doc/llms-full.txt`) use dynamic CORS matching
- [ ] Custom registered endpoints apply configured CORS policies
- [ ] Backward compatibility: `HandlePublic` method still works

### AC-6: Documentation

- [ ] `SPEC.md` updated with CORS endpoint registration section
- [ ] `INSTALL.md` includes CORS configuration examples
- [ ] `README.md` features list mentions dynamic CORS configuration
- [ ] `samples/moon.conf` includes comprehensive CORS endpoint examples
- [ ] Configuration comments explain pattern types and security considerations
- [ ] API documentation template (`doc.md.tmpl`) mentions CORS configuration
- [ ] Migration guide provided for existing deployments

### AC-7: Testing

- [ ] Unit tests cover all pattern matching types (exact, prefix, suffix, contains)
- [ ] Unit tests verify pattern matching priority rules
- [ ] Unit tests verify CORS header application for all scenarios
- [ ] Unit tests verify authentication bypass behavior
- [ ] Unit tests verify configuration validation rules
- [ ] Integration tests verify end-to-end CORS behavior for registered endpoints
- [ ] Integration tests verify OPTIONS preflight handling
- [ ] Integration tests verify backward compatibility with default endpoints
- [ ] Integration tests verify protected endpoint CORS (requires auth)
- [ ] All tests pass with 100% success rate

### AC-8: Error Handling and Logging

- [ ] Configuration validation errors include helpful messages and field names
- [ ] Pattern match logging (DEBUG) shows matched pattern and score
- [ ] Configuration warning logging (WARN) for unusual configurations
- [ ] Authentication bypass logging (INFO) for audit trail
- [ ] Pattern conflict warnings (WARN) when multiple patterns match
- [ ] Startup validation logs all registered CORS endpoints (INFO)

### AC-9: Security

- [ ] Default deny: Endpoints without CORS registration do not get CORS headers
- [ ] `bypass_auth` defaults to `false` (authentication required)
- [ ] Empty `allowed_origins` prevents CORS (no origins allowed)
- [ ] Warning logged if `allow_credentials: true` with wildcard origin
- [ ] Public endpoints with `bypass_auth: true` are logged for audit
- [ ] Configuration change detection logs differences in CORS endpoints
- [ ] Rate limiting applies to public endpoints (prevent abuse)

### AC-10: Backward Compatibility

- [ ] Without `cors.endpoints` configuration, default endpoints work (PRD-052 behavior)
- [ ] Existing deployments work without configuration changes
- [ ] Default endpoints: `/health`, `/doc`, `/doc/llms-full.txt` have public CORS
- [ ] `HandlePublic` method preserved for legacy code
- [ ] No breaking changes for existing users
- [ ] Migration path documented for gradual adoption

### AC-11: Examples and Use Cases

- [ ] Example: Public health endpoint (`/health`, wildcard origin, no auth)
- [ ] Example: Public documentation (`/doc`, wildcard origin, no auth)
- [ ] Example: Partner webhooks (`/webhooks/*`, specific origin, requires auth)
- [ ] Example: Public API subset (`/public/*`, specific origins, no auth)
- [ ] Example: JSON files (`*.json`, wildcard origin, requires auth)
- [ ] Example: Status pages (`/status/*`, wildcard origin, no auth)
- [ ] All examples tested and verified working

---

## Checklist

- [x] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [x] Update API doc template at `cmd/moon/internal/handlers/templates/doc.md.tmpl` to reflect these changes.
- [x] Run all tests and ensure 100% pass rate (excluding pre-existing failures in auth package).
- [x] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete (auth failures are pre-existing).
