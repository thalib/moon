# PRD-059: Remove Legacy X-API-Key Header

## Overview

**Problem Statement:**
Following the implementation of PRD-057 (Unified Authentication Header), Moon currently supports a transitional period where both `Authorization: Bearer <api_key>` and the legacy `X-API-Key: <api_key>` headers are accepted for API key authentication. This transitional support was designed to provide clients time to migrate to the unified header format. After the migration period concludes, the legacy header support must be completely removed to:
- Eliminate technical debt and code complexity
- Complete the authentication standardization effort
- Reduce maintenance burden and potential security vectors
- Ensure all clients are using industry-standard authentication headers

**Context:**
- **PRD-057** implemented unified authentication headers with `Authorization: Bearer` for both JWT and API keys
- Transitional support for `X-API-Key` was included with:
  - Configurable grace period via `legacy_header_support: true` setting
  - Sunset date configuration via `legacy_header_sunset` field
  - Deprecation warnings logged when legacy header is used
  - HTTP `Deprecation`, `Sunset`, and `Link` headers returned to legacy clients
- The transitional period is configurable (default 90 days from PRD-057 implementation)
- Current behavior: Both headers work, with Bearer taking precedence if both present
- Modern REST API standards have converged on `Authorization: Bearer` for all token types

**Solution:**
Completely remove support for the `X-API-Key` header after the transitional period expires. This involves:
1. Removing code paths that handle legacy header detection and processing
2. Removing configuration options related to legacy header support
3. Cleaning up deprecation warning logic and sunset header generation
4. Updating all tests to exclusively use `Authorization: Bearer` header
5. Ensuring JWT authentication remains unaffected (already uses correct header)
6. Documenting the breaking change and final migration deadline

**Benefits:**
- **Reduced Complexity:** Single authentication header code path eliminates conditional logic
- **Improved Security:** Fewer code paths reduce attack surface and potential vulnerabilities
- **Better Maintainability:** Cleaner codebase without legacy support code
- **Standards Compliance:** Full alignment with OAuth 2.0 Bearer token RFC 6750
- **Simplified Testing:** No need to test dual header scenarios
- **Technical Debt Elimination:** Completes the authentication modernization effort
- **Clear Client Expectations:** Single, well-defined authentication method

**Breaking Change:**
This is a **breaking change** for API key clients still using the legacy `X-API-Key` header. Clients that have not migrated to `Authorization: Bearer` during the transitional period will experience authentication failures after this release.

**Migration Timeline:**
- **PRD-057 Release (Version X.Y.0):** Unified header introduced, both headers work, deprecation warnings active
- **90 Days Later:** Increased warning frequency, monitoring of legacy header usage
- **180 Days Later (This PRD):** Complete removal of `X-API-Key` support
- Clients must migrate before the sunset date specified in their deprecation headers

---

## Requirements

### FR-1: Remove Legacy Header Processing Code

**FR-1.1: Authentication Middleware Cleanup**
File: `cmd/moon/internal/middleware/auth.go`

Remove all legacy header processing logic:
- Remove fallback logic that checks `X-API-Key` header when `Authorization` is not present
- Remove conditional branches that handle legacy authentication flow
- Remove deprecation warning generation code
- Remove sunset header injection logic
- Simplify `extractToken()` to only read `Authorization: Bearer` header
- Keep token type detection logic (JWT vs API key via prefix) unchanged

**Before (with legacy support):**
```go
// Extract token from Authorization header or fallback to X-API-Key
token := extractBearerToken(r)
if token == "" && config.LegacyHeaderSupport {
    token = r.Header.Get("X-API-Key")
    if token != "" {
        // Log deprecation warning
        logLegacyHeaderUsage(r)
        // Set response headers for deprecation
        usedLegacyAuth = true
    }
}
```

**After (legacy removed):**
```go
// Extract token from Authorization header only
token := extractBearerToken(r)
if token == "" {
    return errAuthenticationRequired
}
```

**FR-1.2: Remove Deprecation Logging**
Remove all deprecation warning log statements related to legacy header usage:
- Remove `WARN: X-API-Key header is deprecated...` log messages
- Remove metrics tracking legacy header usage counts
- Remove logging of client identification for legacy header users
- Keep standard authentication success/failure logging

**FR-1.3: Remove Sunset Header Generation**
Remove middleware code that adds deprecation-related HTTP headers:
- Remove `Deprecation: true` header injection
- Remove `Sunset: <date>` header injection
- Remove `Link: <doc_url>; rel="deprecation"` header injection
- Remove conditional response header logic based on `usedLegacyAuth` flag

### FR-2: Remove Configuration Options

**FR-2.1: Remove Legacy Support Configuration**
File: `cmd/moon/internal/config/config.go`

Remove configuration fields from `APIKeyConfig` struct:
```go
// REMOVE THESE FIELDS:
LegacyHeaderSupport bool   `yaml:"legacy_header_support"`
LegacyHeaderSunset  string `yaml:"legacy_header_sunset"`
```

**FR-2.2: Remove Configuration Validation**
Remove validation logic for legacy header configuration:
- Remove sunset date parsing and validation
- Remove checks that prevent enabling legacy support after sunset date
- Remove default value initialization for `legacy_header_support` and `legacy_header_sunset`

**FR-2.3: Configuration Migration Handling**
Ensure graceful handling of old configuration files:
- If old config contains `legacy_header_support` or `legacy_header_sunset`, log warning and ignore
- Include message: `WARNING: Configuration fields 'legacy_header_support' and 'legacy_header_sunset' are no longer supported and have been removed. Only Authorization: Bearer header is accepted.`
- Do not fail startup due to deprecated config fields (forward compatibility)

### FR-3: Remove Constants and Helper Functions

**FR-3.1: Remove Deprecation Header Constants**
File: `cmd/moon/internal/middleware/auth.go` or constants file

Remove unused constants:
```go
// REMOVE:
const (
    HeaderDeprecation = "Deprecation"
    HeaderSunset      = "Sunset"
    HeaderLink        = "Link"
)
```

Keep essential constants:
```go
// KEEP:
const (
    HeaderAuthorization = "Authorization"
    BearerPrefix        = "Bearer "
    APIKeyPrefix        = "moon_live_"
)
```

**FR-3.2: Remove Helper Functions**
Remove functions that are no longer needed:
- `logLegacyHeaderUsage()` - logged deprecation warnings
- `addDeprecationHeaders()` - added sunset/deprecation HTTP headers
- `formatSunsetDate()` - formatted sunset date for HTTP header
- `trackLegacyUsage()` - tracked metrics for legacy header usage
- `isLegacyAuthUsed()` - checked if legacy authentication path was taken

### FR-4: Update Tests

**FR-4.1: Remove Legacy Header Tests**
File: `cmd/moon/internal/middleware/auth_test.go`

Remove test cases for legacy authentication:
- `TestLegacyAPIKeyViaXAPIKey` - verified legacy header works
- `TestBothHeadersPresentBearerPrecedence` - verified precedence rules
- `TestDeprecationHeadersPresent` - verified deprecation headers
- `TestLegacyHeaderDisabled` - verified config toggle works
- `TestLegacyUsageMetrics` - verified usage tracking

**FR-4.2: Update Remaining Tests to Use Unified Header**
Ensure all existing authentication tests exclusively use `Authorization: Bearer`:
- Update test helper functions to only set Bearer token header
- Remove test utilities that create `X-API-Key` headers
- Verify no test cases reference the legacy header pattern
- Update test documentation comments to reflect unified authentication

**FR-4.3: Verify Negative Test Cases**
Ensure tests explicitly validate rejection of legacy header:
- Add test case: `X-API-Key` header present without `Authorization` → 401 Unauthorized
- Verify error message clearly states `Authorization: Bearer` is required
- Ensure no hint or reference to legacy header in error messages

**FR-4.4: Integration Test Updates**
File: `scripts/test-auth-migration.sh` (if exists)

Remove or update migration test script:
- Remove tests that verify dual header support
- Remove tests that check deprecation headers
- Keep tests that verify unified authentication works correctly
- Rename script if necessary (no longer testing "migration")

### FR-5: Update Documentation

**FR-5.1: Remove Legacy Header References from SPEC_AUTH.md**
Remove all mentions of legacy authentication:
- Remove "Legacy X-API-Key Header (Deprecated)" sections
- Remove transitional period explanations
- Remove sunset date references
- Remove migration timeline sections
- Remove deprecation warning documentation
- Update authentication flow diagrams to show single header path

Keep only current authentication documentation:
- `Authorization: Bearer <TOKEN>` for all authentication
- Token type detection (JWT vs API key via prefix)
- Error responses for missing/invalid authentication

**FR-5.2: Update API Documentation Template**
File: `cmd/moon/internal/handlers/templates/doc.md.tmpl`

Remove legacy header references:
- Remove any "Migration from X-API-Key" sections
- Remove examples showing `X-API-Key` header
- Remove transitional period notices
- Remove deprecation timeline information

Ensure all examples use unified header:
```bash
# JWT authentication
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  https://api.moon.example.com/data/users:list

# API key authentication
curl -H "Authorization: Bearer moon_live_tlheL3vQr8..." \
  https://api.moon.example.com/data/users:list
```

**FR-5.3: Update Sample Configuration Files**
Files: `samples/moon.conf`, `samples/*.yaml`

Remove legacy configuration options:
```yaml
# REMOVE THESE LINES:
auth:
  apikey:
    legacy_header_support: true
    legacy_header_sunset: "2026-05-08T00:00:00Z"
```

Add comment explaining the breaking change:
```yaml
auth:
  apikey:
    enabled: true
    # Note: As of version X.Y.Z, only Authorization: Bearer header is supported
    # The legacy X-API-Key header has been removed (see PRD-059)
```

**FR-5.4: Update SPEC.md and README.md**
- Remove references to legacy header in authentication sections
- Remove migration guides and transitional period information
- Update authentication examples to show only `Authorization: Bearer`
- Update "Breaking Changes" or "Changelog" section to document removal of legacy header
- Specify exact version where legacy header support was removed

**FR-5.5: Update INSTALL.md**
- Remove any installation notes about legacy header support
- Remove migration instructions for upgrading from pre-PRD-057 versions
- Ensure authentication setup examples use unified header only

### FR-6: Error Message Updates

**FR-6.1: Authentication Required Error**
Update error message when no authentication header is present:

```json
{
  "error": "authentication_required",
  "message": "Authorization header required. Use: Authorization: Bearer <token>",
  "status": 401
}
```

Ensure no mention of `X-API-Key` as an alternative.

**FR-6.2: Invalid Token Format Error**
Keep existing error for invalid token format:

```json
{
  "error": "invalid_token_format",
  "message": "Token must be a valid JWT or API key (starting with moon_live_)",
  "status": 401
}
```

**FR-6.3: Error Documentation**
Update error code documentation to remove legacy header references:
- Remove examples showing `X-API-Key` in error scenarios
- Remove "migration tips" from error documentation
- Keep clear examples of correct authentication format

### FR-7: Code Quality and Cleanup

**FR-7.1: Remove Dead Code**
Perform comprehensive search and removal:
- Search codebase for `X-API-Key` string literal
- Search for `legacy` in authentication-related files
- Search for `sunset` in authentication-related files
- Search for `deprecation` in authentication-related files
- Remove all related code blocks, comments, and documentation

**FR-7.2: Simplify Authentication Flow**
With legacy support removed, refactor for clarity:
- Simplify conditional branches in authentication middleware
- Remove unnecessary flag variables (e.g., `usedLegacyAuth`)
- Consolidate error handling paths
- Update code comments to reflect current behavior only
- Remove historical context comments about migration

**FR-7.3: Update Code Documentation**
Update function and method documentation:
```go
// BEFORE:
// extractToken extracts the authentication token from the request.
// It first checks the Authorization: Bearer header, and during the
// transitional period, falls back to X-API-Key header if configured.

// AFTER:
// extractToken extracts the authentication token from the Authorization: Bearer header.
// Returns empty string if header is not present or malformed.
```

### FR-8: Version and Release Notes

**FR-8.1: Update ROADMAP.md**
- Mark PRD-059 as completed
- Note the breaking change in version where this is released
- Update authentication section to reflect completion of modernization

**FR-8.2: Create Migration Advisory**
Create clear communication about the breaking change:

**Release Notes Entry:**
```markdown
## Breaking Changes

### Removed Legacy X-API-Key Header Support (PRD-059)

As of version X.Y.Z, the legacy `X-API-Key` header is no longer supported for API key authentication. This completes the authentication modernization effort started in PRD-057.

**What Changed:**
- Only `Authorization: Bearer <token>` header is accepted
- API keys must be sent as: `Authorization: Bearer moon_live_<key>`
- JWT tokens continue to use: `Authorization: Bearer <jwt_token>`

**Action Required:**
If you are still using `X-API-Key` header, you MUST update your API clients before upgrading to this version. Requests using the legacy header will receive `401 Unauthorized` errors.

**Migration:**
Replace:
```bash
curl -H "X-API-Key: moon_live_abc123..." https://api.example.com/data/users:list
```

With:
```bash
curl -H "Authorization: Bearer moon_live_abc123..." https://api.example.com/data/users:list
```

**Timeline:**
- PRD-057 (v1.0.0): Unified header introduced, both headers supported
- 90 days: Deprecation warnings increased
- 180 days (v2.0.0): Legacy header removed (this release)

For assistance, contact: support@moon.example.com
```

**FR-8.3: Update TODO.md**
Remove completed migration tasks:
- Remove "Monitor legacy header usage" tasks
- Remove "Remove legacy header support after grace period" tasks
- Add completion date for PRD-059

---

## Acceptance

### AC-1: Code Removal

- [ ] All legacy `X-API-Key` header processing code removed from `cmd/moon/internal/middleware/auth.go`
- [ ] Fallback logic to check legacy header removed
- [ ] Deprecation warning logging code removed
- [ ] Sunset header injection code removed
- [ ] Legacy usage metrics tracking code removed
- [ ] Helper functions for legacy authentication removed
- [ ] Deprecation header constants removed
- [ ] Configuration fields `legacy_header_support` and `legacy_header_sunset` removed
- [ ] Configuration validation for legacy settings removed
- [ ] No references to `X-API-Key` remain in source code (except in comments explaining the removal)

### AC-2: Backward Compatibility Handling

- [ ] Startup succeeds even if old config file contains deprecated `legacy_header_support` field
- [ ] Warning logged when deprecated config fields detected
- [ ] No breaking changes for clients already using `Authorization: Bearer` header
- [ ] JWT authentication completely unaffected by legacy header removal
- [ ] API key authentication via Bearer token continues working correctly

### AC-3: Testing

- [ ] All tests for legacy header behavior removed
- [ ] Test case added verifying `X-API-Key` header is rejected with 401
- [ ] Test case verifies error message does not mention legacy header as valid option
- [ ] All remaining authentication tests use `Authorization: Bearer` exclusively
- [ ] Test helper functions updated to only create Bearer token headers
- [ ] Integration tests pass with 100% success rate
- [ ] No test cases reference `X-API-Key` header except negative rejection tests
- [ ] Backward compatibility tests for transitional period removed

### AC-4: Documentation Updates

- [ ] All references to `X-API-Key` header removed from `SPEC_AUTH.md`
- [ ] Migration guide sections removed from documentation
- [ ] Transitional period explanations removed
- [ ] API documentation template (`doc.md.tmpl`) contains no legacy header references
- [ ] All curl examples use `Authorization: Bearer` format exclusively
- [ ] Sample configuration files have legacy options removed
- [ ] `SPEC.md` updated to reflect legacy header removal
- [ ] `README.md` updated to remove legacy authentication mentions
- [ ] `INSTALL.md` updated to remove migration instructions
- [ ] Breaking change documented in release notes

### AC-5: Error Handling

- [ ] Request with `X-API-Key` header (no Authorization header) returns 401
- [ ] Error message clearly states `Authorization: Bearer` is required
- [ ] Error message does NOT mention `X-API-Key` as alternative
- [ ] Request with both `X-API-Key` and valid Bearer token succeeds (legacy header ignored)
- [ ] No deprecation headers (`Sunset`, `Deprecation`, `Link`) included in responses
- [ ] Error responses match updated error message specifications

### AC-6: Configuration

- [ ] Deprecated config fields (`legacy_header_support`, `legacy_header_sunset`) are ignored if present
- [ ] Warning logged when deprecated config fields detected at startup
- [ ] Configuration validation does not fail due to deprecated fields
- [ ] Sample configs reflect removal of legacy options
- [ ] Config documentation updated to remove legacy fields

### AC-7: Code Quality

- [ ] Code search for `X-API-Key` returns only comments and documentation explaining removal
- [ ] Code search for `legacy` in auth files returns no active code paths
- [ ] Code search for `sunset` returns no authentication-related code
- [ ] Authentication middleware simplified with fewer conditional branches
- [ ] Function documentation updated to remove historical context
- [ ] Code comments reflect current authentication behavior only
- [ ] No dead code or unused variables related to legacy authentication

### AC-8: Release Management

- [ ] Breaking change documented in `ROADMAP.md`
- [ ] Version number incremented appropriately (major version bump recommended)
- [ ] Release notes include clear migration instructions
- [ ] Communication plan prepared for notifying API consumers
- [ ] Migration advisory document created with timeline and support contact
- [ ] `TODO.md` updated to remove legacy header migration tasks

### AC-9: Functional Verification

- [ ] JWT authentication via `Authorization: Bearer <jwt>` works correctly
- [ ] API key authentication via `Authorization: Bearer moon_live_<key>` works correctly
- [ ] Token type detection (JWT vs API key) functions correctly
- [ ] Missing `Authorization` header returns 401 with appropriate error
- [ ] Invalid token format returns 401 with appropriate error
- [ ] Valid authentication succeeds and returns expected data
- [ ] All protected endpoints enforce authentication correctly
- [ ] Public endpoints (if any) continue to work without authentication

### AC-10: Test Coverage

- [ ] All authentication middleware tests pass
- [ ] All handler tests pass
- [ ] All config tests pass
- [ ] Integration tests pass with 100% success rate
- [ ] No test failures related to legacy header removal
- [ ] Code coverage for authentication module maintained or improved
- [ ] Edge cases covered: empty header, malformed header, wrong scheme (e.g., `Basic`)

---

## Checklist

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [ ] Update API doc template at `cmd/moon/internal/handlers/templates/doc.md.tmpl` to reflect these changes.
- [ ] Run all tests and ensure 100% pass rate.
- [ ] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete.

---

## Implementation Notes

### Files to Modify

1. **Authentication Middleware:**
   - `cmd/moon/internal/middleware/auth.go` - Remove legacy header processing

2. **Configuration:**
   - `cmd/moon/internal/config/config.go` - Remove legacy config fields

3. **Tests:**
   - `cmd/moon/internal/middleware/auth_test.go` - Remove legacy tests, add rejection tests

4. **Documentation:**
   - `SPEC_AUTH.md` - Remove legacy header documentation
   - `SPEC.md` - Update authentication section
   - `README.md` - Remove legacy references
   - `INSTALL.md` - Remove migration instructions
   - `cmd/moon/internal/handlers/templates/doc.md.tmpl` - Remove legacy examples

5. **Samples:**
   - `samples/moon.conf` - Remove legacy config options
   - Other config samples in `samples/` directory

6. **Project Management:**
   - `ROADMAP.md` - Mark PRD-059 as completed
   - `TODO.md` - Remove legacy migration tasks

### Removal Checklist

Search and remove all occurrences of:
- [ ] `X-API-Key` (in code, not documentation explaining removal)
- [ ] `LegacyHeaderSupport`
- [ ] `LegacyHeaderSunset`
- [ ] `HeaderDeprecation`
- [ ] `HeaderSunset`
- [ ] `HeaderLink` (if only used for deprecation)
- [ ] `logLegacyHeaderUsage`
- [ ] `addDeprecationHeaders`
- [ ] `formatSunsetDate`
- [ ] `trackLegacyUsage`
- [ ] `isLegacyAuthUsed`
- [ ] `usedLegacyAuth` (variable)

### Testing Strategy

1. **Unit Tests:**
   - Remove ~5 legacy authentication test cases
   - Add 1-2 test cases for legacy header rejection
   - Update ~10 existing tests to ensure Bearer-only usage

2. **Integration Tests:**
   - Run full test suite to ensure no regressions
   - Verify JWT authentication unaffected
   - Verify API key Bearer authentication works
   - Verify legacy header properly rejected

3. **Manual Testing:**
   - Test with curl using various header combinations
   - Verify error messages are clear and accurate
   - Verify no deprecation headers in responses
   - Verify old config files don't break startup

### Rollout Strategy

1. **Pre-Release:**
   - Communicate breaking change to all API consumers
   - Provide migration deadline (before release)
   - Offer support for migration assistance

2. **Release:**
   - Increment major version number (e.g., 1.x.x → 2.0.0)
   - Publish comprehensive release notes
   - Update public documentation immediately

3. **Post-Release:**
   - Monitor for authentication failures
   - Provide rapid support for migration issues
   - Consider emergency hotfix path if needed

### Risk Mitigation

**Risk:** Clients not migrated before release experience auth failures
**Mitigation:**
- Clear communication 90 and 30 days before release
- Provide migration guide with code examples
- Offer extended support during transition
- Consider delayed removal if significant clients unmigrated

**Risk:** Incomplete code removal leaves dead code
**Mitigation:**
- Use comprehensive grep/search for all related terms
- Code review focused on cleanup completeness
- Automated linting to detect unused code

**Risk:** Breaking JWT authentication during cleanup
**Mitigation:**
- JWT already uses correct header, no changes needed
- Comprehensive test suite for JWT authentication
- Integration tests verify end-to-end flows

---

## Success Criteria

This PRD is considered successfully implemented when:

1. **Legacy code completely removed:** No references to `X-API-Key` header in active code paths
2. **Tests all pass:** 100% test pass rate with updated test suite
3. **Documentation updated:** No legacy header references in user-facing docs
4. **Clear error messages:** 401 errors guide users to correct header format
5. **JWT unaffected:** JWT authentication continues working without issues
6. **API key Bearer works:** API key authentication via Bearer token functions correctly
7. **Breaking change documented:** Release notes clearly communicate the change
8. **Configuration cleanup:** Legacy config fields removed or safely ignored

**Completion Criteria:**
- All acceptance criteria checked and verified
- All checklist items completed
- Code review approved
- QA testing passed
- Release notes drafted
- Documentation published
