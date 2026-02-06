# PRD-044-D: Authorization Middleware & Rate Limiting

## Overview

Implement authorization middleware for role-based access control (RBAC) and rate limiting middleware for protecting against abuse. This layer sits between authentication and request handlers, enforcing permission checks and request limits based on user/API key identity.

### Problem Statement

After authentication identifies the user or API key, the system must enforce:
- Role-based access control (admin vs user)
- Write permission checks for user role
- Per-entity rate limiting to prevent abuse
- Audit logging for authorization failures and rate limit violations

### Dependencies

- **PRD-044-A:** Core authentication (JWT/API key middleware, user context)
- **PRD-044-B:** User management endpoints (requires admin authorization)
- **PRD-044-C:** API key management endpoints (requires admin authorization)

---

## Requirements

### FR-1: Authorization Middleware

**FR-1.1: Middleware Structure**
```go
type AuthorizationMiddleware struct {
    // No configuration needed - reads from request context
}

func NewAuthorizationMiddleware() *AuthorizationMiddleware {
    return &AuthorizationMiddleware{}
}
```

**FR-1.2: Authorization Logic**
```go
func (m *AuthorizationMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get auth entity from context (set by authentication middleware)
            authEntity := GetAuthEntityFromContext(r.Context())
            
            if authEntity == nil {
                // Should not happen if auth middleware ran first
                writeError(w, 401, "MISSING_AUTH", "Authentication required")
                return
            }
            
            // Check role
            if authEntity.Role != role && role != "" {
                auditLog("AUTHZ_FAILURE", authEntity, r.URL.Path, "insufficient role")
                writeError(w, 403, "ADMIN_REQUIRED", "This action requires admin role")
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

**FR-1.3: Write Permission Check**
```go
func (m *AuthorizationMiddleware) RequireWrite() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authEntity := GetAuthEntityFromContext(r.Context())
            
            if authEntity == nil {
                writeError(w, 401, "MISSING_AUTH", "Authentication required")
                return
            }
            
            // Admin always has write permission
            if authEntity.Role == "admin" {
                next.ServeHTTP(w, r)
                return
            }
            
            // User must have can_write flag
            if authEntity.Role == "user" && !authEntity.CanWrite {
                auditLog("AUTHZ_FAILURE", authEntity, r.URL.Path, "write permission required")
                writeError(w, 403, "WRITE_PERMISSION_REQUIRED", "Write access not granted for this user")
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

---

### FR-2: Endpoint Protection Matrix

**FR-2.1: Public Endpoints (No Auth)**
- `GET /health`
- `GET /doc/`
- `GET /doc/llms-full.txt`

**FR-2.2: Public Auth Endpoints (No Role Check)**
- `POST /auth:login`
- `POST /auth:refresh`

**FR-2.3: Authenticated Endpoints (Any Role)**
- `POST /auth:logout`
- `GET /auth:me`
- `POST /auth:me`
- `GET /collections:list`
- `GET /collections:get`
- `GET /{collection}:list`
- `GET /{collection}:get`
- `GET /{collection}:count`
- `GET /{collection}:sum`
- `GET /{collection}:avg`
- `GET /{collection}:min`
- `GET /{collection}:max`

**FR-2.4: Admin-Only Endpoints**
- `GET /users:list`
- `GET /users:get`
- `POST /users:create`
- `POST /users:update`
- `POST /users:destroy`
- `GET /apikeys:list`
- `GET /apikeys:get`
- `POST /apikeys:create`
- `POST /apikeys:update`
- `POST /apikeys:destroy`
- `POST /collections:create`
- `POST /collections:update`
- `POST /collections:destroy`

**FR-2.5: Write Permission Required**
- `POST /{collection}:create`
- `POST /{collection}:update`
- `POST /{collection}:destroy`

**Logic:**
- Admin role: Always has write permission (bypasses `can_write` check)
- User role with `can_write: false`: Read-only access
- User role with `can_write: true`: Can write data but not manage collections

---

### FR-3: Rate Limiting Middleware

**FR-3.1: Rate Limiter Structure**
```go
type RateLimiter struct {
    userLimit   int           // requests per minute for users
    apikeyLimit int           // requests per minute for API keys
    loginLimit  int           // login attempts per window
    loginWindow time.Duration // login attempt window
    
    // In-memory storage (use sync.Map for concurrency)
    buckets sync.Map // entity_id -> *TokenBucket
}

type TokenBucket struct {
    capacity  int       // max tokens
    tokens    int       // current tokens
    refillAt  time.Time // when to refill
    mu        sync.Mutex
}
```

**FR-3.2: Rate Limit Configuration**
```go
type RateLimitConfig struct {
    UserRPM         int           // default: 100
    APIKeyRPM       int           // default: 1000
    LoginAttempts   int           // default: 5
    LoginWindow     time.Duration // default: 15 minutes
}
```

**FR-3.3: Token Bucket Algorithm**
```go
func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    now := time.Now()
    
    // Refill bucket if window passed
    if now.After(tb.refillAt) {
        tb.tokens = tb.capacity
        tb.refillAt = now.Add(time.Minute)
    }
    
    // Check if tokens available
    if tb.tokens > 0 {
        tb.tokens--
        return true
    }
    
    return false
}

func (tb *TokenBucket) Remaining() int {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    now := time.Now()
    if now.After(tb.refillAt) {
        return tb.capacity
    }
    return tb.tokens
}

func (tb *TokenBucket) ResetAt() time.Time {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    return tb.refillAt
}
```

**FR-3.4: Rate Limit Middleware**
```go
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authEntity := GetAuthEntityFromContext(r.Context())
        
        if authEntity == nil {
            // No auth entity - skip rate limiting (will fail at auth middleware)
            next.ServeHTTP(w, r)
            return
        }
        
        // Determine rate limit
        limit := rl.userLimit
        if authEntity.IsAPIKey {
            limit = rl.apikeyLimit
        }
        
        // Get or create token bucket
        bucketKey := authEntity.ID
        bucketI, _ := rl.buckets.LoadOrStore(bucketKey, &TokenBucket{
            capacity: limit,
            tokens:   limit,
            refillAt: time.Now().Add(time.Minute),
        })
        bucket := bucketI.(*TokenBucket)
        
        // Check rate limit
        if !bucket.Allow() {
            auditLog("RATE_LIMIT_EXCEEDED", authEntity, r.URL.Path, "too many requests")
            
            // Add rate limit headers
            w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
            w.Header().Set("X-RateLimit-Remaining", "0")
            w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(bucket.ResetAt().Unix(), 10))
            
            writeError(w, 429, "RATE_LIMIT_EXCEEDED", 
                fmt.Sprintf("Too many requests. Limit: %d per minute", limit))
            return
        }
        
        // Add rate limit headers
        w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
        w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(bucket.Remaining()))
        w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(bucket.ResetAt().Unix(), 10))
        
        next.ServeHTTP(w, r)
    })
}
```

---

### FR-4: Login Rate Limiting

**FR-4.1: Login Rate Limiter**
```go
type LoginRateLimiter struct {
    attempts map[string]*LoginAttempts // key -> attempts
    mu       sync.Mutex
}

type LoginAttempts struct {
    count     int
    expiresAt time.Time
}

func (lrl *LoginRateLimiter) Allow(key string) bool {
    lrl.mu.Lock()
    defer lrl.mu.Unlock()
    
    now := time.Now()
    
    // Clean up expired entries
    if attempts, ok := lrl.attempts[key]; ok {
        if now.After(attempts.expiresAt) {
            delete(lrl.attempts, key)
        }
    }
    
    // Get or create attempts
    if _, ok := lrl.attempts[key]; !ok {
        lrl.attempts[key] = &LoginAttempts{
            count:     0,
            expiresAt: now.Add(15 * time.Minute),
        }
    }
    
    attempts := lrl.attempts[key]
    
    // Check limit
    if attempts.count >= 5 {
        return false
    }
    
    attempts.count++
    return true
}
```

**FR-4.2: Login Tracking Key**
```
{ip_address}:{username}
```

Example: `192.168.1.100:admin`

**FR-4.3: Integration with Login Handler**
```go
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Build rate limit key
    ip := getClientIP(r)
    key := fmt.Sprintf("%s:%s", ip, req.Username)
    
    // Check login rate limit
    if !h.loginRateLimiter.Allow(key) {
        auditLog("LOGIN_RATE_LIMIT", ip, req.Username, "too many attempts")
        writeError(w, 429, "LOGIN_ATTEMPTS_EXCEEDED", 
            "Too many failed login attempts. Try again in 15 minutes")
        return
    }
    
    // ... rest of login logic
}
```

---

### FR-5: Middleware Execution Order

**FR-5.1: Server Middleware Stack**
```go
// Server setup in server.go
func (s *Server) setupMiddleware() http.Handler {
    var handler http.Handler = s.mux
    
    // Wrap in reverse order (last middleware wraps first)
    handler = s.errorHandlerMiddleware(handler)     // 7. Catch panics
    handler = s.loggingMiddleware(handler)          // 6. Log requests
    handler = s.validationMiddleware(handler)       // 5. Validate input
    handler = s.authorizationMiddleware(handler)    // 4. Check permissions
    handler = s.rateLimitMiddleware(handler)        // 3. Rate limiting
    handler = s.authenticationMiddleware(handler)   // 2. Authenticate
    handler = s.corsMiddleware(handler)             // 1. CORS headers
    
    return handler
}
```

**FR-5.2: Execution Flow**
1. CORS: Add headers for cross-origin requests
2. Authentication: Extract and validate JWT/API key, add entity to context
3. Rate Limiting: Check per-entity rate limits
4. Authorization: Check role and permissions
5. Validation: Validate request body/params (if applicable)
6. Logging: Log request details
7. Handler: Execute business logic
8. Error Handling: Catch panics and format errors

---

### FR-6: Audit Logging

**FR-6.1: Authorization Failures**
```
WARN: AUTHZ_FAILURE entity_id={ulid} entity_type={user|apikey} endpoint={path} reason={insufficient_role|write_permission_required}
```

**FR-6.2: Rate Limit Violations**
```
WARN: RATE_LIMIT_EXCEEDED entity_id={ulid} entity_type={user|apikey} endpoint={path} limit={requests_per_minute}
```

**FR-6.3: Login Rate Limit**
```
WARN: LOGIN_RATE_LIMIT ip={ip_address} username={username} attempts={count}
```

---

## Acceptance Criteria

### AC-1: Admin-Only Endpoint Protection

**Verification:**
- [ ] Admin can access all admin-only endpoints
- [ ] User cannot access admin-only endpoints
- [ ] Returns 403 for non-admin access

**Test:**
```bash
# Login as admin
ADMIN_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"AdminPass123"}' | jq -r '.access_token')

# Login as user
USER_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"user1","password":"UserPass123"}' | jq -r '.access_token')

# Test admin access
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/users:list"
# Expected: 200 OK

# Test user access (should fail)
curl -H "Authorization: Bearer $USER_TOKEN" \
  "http://localhost:6006/users:list"
# Expected: 403 Forbidden
```

### AC-2: Write Permission Enforcement

**Verification:**
- [ ] Admin can write data
- [ ] User with `can_write: true` can write data
- [ ] User with `can_write: false` cannot write data
- [ ] User cannot create/update/destroy collections (regardless of can_write)

**Test:**
```bash
# Create read-only user
READONLY_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"readonly","password":"ReadPass123"}' | jq -r '.access_token')

# Create read-write user
READWRITE_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"readwrite","password":"WritePass123"}' | jq -r '.access_token')

# Read-only user can read
curl -H "Authorization: Bearer $READONLY_TOKEN" \
  "http://localhost:6006/products:list"
# Expected: 200 OK

# Read-only user cannot write
curl -X POST http://localhost:6006/products:create \
  -H "Authorization: Bearer $READONLY_TOKEN" \
  -d '{"name":"Product"}'
# Expected: 403 Forbidden with WRITE_PERMISSION_REQUIRED

# Read-write user can write
curl -X POST http://localhost:6006/products:create \
  -H "Authorization: Bearer $READWRITE_TOKEN" \
  -d '{"name":"Product"}'
# Expected: 200 OK

# Read-write user cannot manage collections
curl -X POST http://localhost:6006/collections:create \
  -H "Authorization: Bearer $READWRITE_TOKEN" \
  -d '{"name":"test"}'
# Expected: 403 Forbidden with ADMIN_REQUIRED
```

### AC-3: Rate Limiting

**Verification:**
- [ ] Users limited to 100 req/min
- [ ] API keys limited to 1000 req/min
- [ ] Returns 429 when limit exceeded
- [ ] Rate limit headers present in all responses
- [ ] Bucket refills after 1 minute

**Test:**
```bash
# Test user rate limit (100 req/min)
for i in {1..101}; do
  RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null \
    -H "Authorization: Bearer $USER_TOKEN" \
    "http://localhost:6006/collections:list")
  
  if [ $i -le 100 ]; then
    [ "$RESPONSE" = "200" ] || echo "Expected 200 for request $i"
  else
    [ "$RESPONSE" = "429" ] || echo "Expected 429 for request $i"
  fi
done

# Check rate limit headers
curl -i -H "Authorization: Bearer $USER_TOKEN" \
  "http://localhost:6006/collections:list" | grep "X-RateLimit"

# Expected:
# X-RateLimit-Limit: 100
# X-RateLimit-Remaining: 99
# X-RateLimit-Reset: {unix_timestamp}

# Wait 1 minute and verify bucket refills
sleep 61
curl -H "Authorization: Bearer $USER_TOKEN" \
  "http://localhost:6006/collections:list"
# Expected: 200 OK (bucket refilled)
```

### AC-4: Login Rate Limiting

**Verification:**
- [ ] Failed logins limited to 5 attempts per 15 min
- [ ] Rate limit tracked per IP + username combination
- [ ] Returns 429 after 5 failures
- [ ] Successful login doesn't count toward limit

**Test:**
```bash
# Attempt 5 failed logins
for i in {1..5}; do
  curl -X POST http://localhost:6006/auth:login \
    -d '{"username":"admin","password":"wrong"}'
  # Expected: 401 Unauthorized
done

# 6th attempt should be rate limited
curl -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"wrong"}'
# Expected: 429 Too Many Requests with LOGIN_ATTEMPTS_EXCEEDED

# Try different username from same IP (should work)
curl -X POST http://localhost:6006/auth:login \
  -d '{"username":"user1","password":"UserPass123"}'
# Expected: 200 OK (different rate limit key)

# Wait 15 minutes and retry
sleep 901
curl -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"AdminPass123"}'
# Expected: 200 OK (rate limit expired)
```

### AC-5: Middleware Execution Order

**Verification:**
- [ ] CORS runs before authentication
- [ ] Authentication runs before rate limiting
- [ ] Rate limiting runs before authorization
- [ ] Authorization runs before handler

**Test:**
```go
// Test middleware order with mock handlers
func TestMiddlewareOrder(t *testing.T) {
    order := []string{}
    
    // Create mock middlewares that record execution order
    corsMiddleware := func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            order = append(order, "cors")
            next.ServeHTTP(w, r)
        })
    }
    
    authMiddleware := func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            order = append(order, "auth")
            next.ServeHTTP(w, r)
        })
    }
    
    rateLimitMiddleware := func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            order = append(order, "ratelimit")
            next.ServeHTTP(w, r)
        })
    }
    
    authzMiddleware := func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            order = append(order, "authz")
            next.ServeHTTP(w, r)
        })
    }
    
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        order = append(order, "handler")
    })
    
    // Wrap in correct order
    wrapped := authzMiddleware(rateLimitMiddleware(authMiddleware(corsMiddleware(handler))))
    
    // Execute request
    req := httptest.NewRequest("GET", "/test", nil)
    rec := httptest.NewRecorder()
    wrapped.ServeHTTP(rec, req)
    
    // Verify execution order
    expected := []string{"cors", "auth", "ratelimit", "authz", "handler"}
    assert.Equal(t, expected, order)
}
```

### AC-6: Audit Logging

**Verification:**
- [ ] Authorization failures logged
- [ ] Rate limit violations logged
- [ ] Login rate limit violations logged
- [ ] Logs include entity ID and endpoint

**Test:**
```bash
# Trigger various audit events
# Check logs

grep "AUTHZ_FAILURE" /var/log/moon/main.log
grep "RATE_LIMIT_EXCEEDED" /var/log/moon/main.log
grep "LOGIN_RATE_LIMIT" /var/log/moon/main.log

# Verify log format
# Example: WARN: AUTHZ_FAILURE entity_id=01ARZ... entity_type=user endpoint=/users:list reason=insufficient_role
```

---

## Implementation Checklist

- [ ] Create `cmd/moon/internal/middleware/authorization.go`
- [ ] Implement `AuthorizationMiddleware` struct
- [ ] Implement `RequireRole` middleware
- [ ] Implement `RequireWrite` middleware
- [ ] Create `cmd/moon/internal/middleware/ratelimit.go`
- [ ] Implement `RateLimiter` struct with token bucket
- [ ] Implement `TokenBucket` with refill logic
- [ ] Implement rate limit middleware
- [ ] Implement login rate limiter
- [ ] Add rate limit headers to responses
- [ ] Add authorization audit logging
- [ ] Add rate limit audit logging
- [ ] Update server middleware stack with correct order
- [ ] Apply authorization to protected endpoints
- [ ] Apply write permission checks to data modification endpoints
- [ ] Write unit tests for token bucket algorithm
- [ ] Write unit tests for authorization logic
- [ ] Write unit tests for rate limiting
- [ ] Write integration tests for endpoint protection
- [ ] Test middleware execution order
- [ ] Test rate limit bucket refill
- [ ] Test login rate limiting

---

## Related PRDs

- [PRD-044: Authentication System](044-authentication-system.md) - Parent PRD
- [PRD-044-A: Core Authentication](044-A-core-authentication.md) - Authentication middleware
- [PRD-044-B: User Management](044-B-user-management.md) - Admin-only endpoints
- [PRD-044-C: API Key Management](044-C-apikey-management.md) - Admin-only endpoints

---

## References

- [SPEC_AUTH.md](../SPEC_AUTH.md) - Authorization and rate limiting specification
- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket) - Rate limiting algorithm
