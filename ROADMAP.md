## SPEC Compliance Audit (2026-01-29)

### Missing from Implementation (SPEC-required)
- **Auth middleware enforcement**: JWT and API key checks exist but are not applied before handlers. The SPEC requires a security layer that enforces allow/deny on every request.
- **Dynamic OpenAPI endpoint**: OpenAPI generation exists, but there is no HTTP endpoint serving the live spec. The SPEC requires dynamic OpenAPI that reflects the in-memory registry and includes auth requirements and example payloads.
- **Collection schema updates**: `/collections:update` only supports adding columns. The SPEC requires add/remove/rename support.


## Moon Server Validation Scripts - Requirements Specification

## Document Overview

**Purpose:** Define authoritative requirements for implementing comprehensive validation shell scripts for the Moon server application.

**Scope:** All user-facing functionality, API endpoints, configuration handling, error cases, edge conditions, and operational modes as defined in SPEC.md.

**Target Audience:** Engineers implementing validation scripts without further clarification.

**Version:** 1.0
**Date:** 2026-01-30

## Executive Summary

Successfully created comprehensive requirements document for Moon server validation scripts covering:

- **15 new validation scripts** to implement
- **5 existing scripts** to enhance
- **200+ test cases** across 14 feature areas
- Complete coverage of SPEC.md functionality

### Key Deliverables

1. Configuration validation (10 test cases)
2. Startup validation (11 test cases) 
3. Health endpoint validation (5 test cases)
4. Collection lifecycle (21 test cases)
5. Data operations (33 test cases)
6. Aggregation endpoints (22 test cases)
7. Error handling (10 test cases)
8. Advanced queries (16 test cases)
9. Concurrency testing (6 test cases)
10. Database dialects (8 test cases)
11. ULID validation (6 test cases)
12. Integration testing (full workflow)
13. Negative testing (comprehensive)
14. Performance validation (7 test cases)

### Implementation Priority

**Phase 1 - Core (High Priority):**
- lib/utils.sh (foundation)
- config-validation.sh
- startup-validation.sh
- health-validation.sh
- error-handling.sh

**Phase 2 - API (High Priority):**
- collection-lifecycle.sh
- data-operations.sh
- aggregation-validation.sh

**Phase 3 - Advanced (Medium Priority):**
- query-advanced.sh
- ulid-validation.sh
- concurrency-validation.sh

**Phase 4 - Comprehensive (Medium/Low Priority):**
- integration-full.sh
- negative-tests.sh
- dialect-validation.sh
- performance-validation.sh

**Phase 5 - Orchestration:**
- run-all-validations.sh
- Enhance existing scripts

## Document Content Summary

The full PLAN5.md document contains:

1. **General Requirements** - Shell standards, exit codes, error handling, logging
2. **Script Standards** - Template, utilities library, best practices
3. **Test Case Specifications** - Detailed requirements for each test
4. **Success Criteria** - Clear acceptance criteria per script
5. **Edge Cases** - Boundary conditions and failure modes
6. **Implementation Guidelines** - Code templates and patterns
7. **Validation Matrix** - Coverage mapping
8. **Appendices** - Test data, environment variables, error catalog

## Key Features

- **Strict Mode**: All scripts use `set -euo pipefail`
- **Deterministic**: Tests are repeatable without side effects
- **Portable**: Works on Linux, macOS, Windows (WSL/Git Bash)
- **Comprehensive**: Covers all SPEC.md functionality
- **Maintainable**: Consistent patterns and style
- **Well-Documented**: Clear purpose and test cases per script

## Test Coverage Areas

| Area | Scripts | Test Cases |
|------|---------|------------|
| Configuration | 1 | 10 |
| Startup | 1 | 11 |
| Health | 1 | 5 |
| Collections | 1 | 21 |
| Data Ops | 1 | 33 |
| Aggregation | 1 | 22 |
| Errors | 1 | 10 |
| Queries | 1 | 16 |
| Concurrency | 1 | 6 |
| Dialects | 1 | 8 |
| ULID | 1 | 6 |
| Integration | 1 | Full |
| Negative | 1 | All |
| Performance | 1 | 7 |
| **Total** | **15** | **200+** |

## Next Steps

1. Review and approve this requirements document
2. Begin Phase 1 implementation (core validation scripts)
3. Create lib/utils.sh common utilities
4. Implement high-priority validation scripts
5. Test and validate each script as implemented
6. Progress through remaining phases
7. Create master orchestration script

