# PRD-043: Remove USAGE.md and Consolidate Documentation

> **Note:** This PRD has been superseded. The `/doc/md` endpoint referenced in this document has been moved to `/doc/llms-full.txt` with no backward compatibility. See current implementation in SPEC.md.

## Overview

### Problem Statement

The project currently maintains duplicate API documentation across multiple files:
- `USAGE.md`: Comprehensive usage guide with API reference, examples, and troubleshooting
- `/doc/` endpoint: HTML API documentation generated dynamically
- `/doc/md` endpoint: Markdown API documentation for AI agents and terminal viewing

This duplication violates the project's principle of keeping documentation lean, clean, and avoiding redundancy. The dynamic documentation endpoints (`/doc/` and `/doc/md`) were specifically created to provide up-to-date, schema-aware API documentation, rendering the static `USAGE.md` file obsolete.

### Context and Background

- Moon follows the principle "SPEC.md is the only source of truth" (from AGENTS.md)
- The project emphasizes simplicity: "Keep code, configuration, and docs lean, simple, and clean" (from AGENTS.md)
- PRD-036 and PRD-037 introduced dynamic documentation endpoints (`/doc/` and `/doc/md`)
- PRD-039 added copy button functionality to code blocks in the documentation
- The `/doc/` endpoint provides complete API reference with quickstart guide, filtering, sorting, pagination, authentication requirements, and error formats
- The `/doc/md` endpoint provides the same content in Markdown format for AI agents and terminal viewing
- `README.md` should focus on project overview and features only (per AGENTS.md)

### High-Level Solution Summary

Remove the `USAGE.md` file entirely and update `README.md` to reference the dynamic documentation endpoints (`/doc/` and `/doc/md`) instead of linking to static usage documentation. This consolidates all API documentation into a single, always-up-to-date source and reduces maintenance burden.

---

## Requirements

### Functional Requirements

#### FR-1: Remove USAGE.md File

**FR-1.1: File Deletion**
- Delete `USAGE.md` from the repository root
- Remove all unique content that is not already covered in `/doc/` or `/doc/md`

**FR-1.2: Content Migration (if needed)**
- Verify that all essential information from `USAGE.md` exists in the documentation template
- If any critical information is missing from `/doc/md`, update the documentation template first before removing `USAGE.md`
- Ensure `/doc/` and `/doc/md` endpoints include:
  - Complete API reference with all endpoints
  - Quickstart guide with copy-pasteable examples
  - Filtering, sorting, and pagination documentation
  - Authentication requirements
  - Error response formats
  - ULID format documentation
  - API pattern explanation (`:action` custom actions)
  - URL prefix configuration guidance

#### FR-2: Update README.md

**FR-2.1: Remove USAGE.md References**
- Remove all references to `USAGE.md` in the "Documentation & Support" section
- Remove any usage or example sections that duplicate `/doc/` content

**FR-2.2: Add Documentation Endpoint References**
- Add clear guidance on accessing the dynamic documentation endpoints
- Include instructions for both HTML (`/doc/`) and Markdown (`/doc/md`) formats
- Provide examples for viewing documentation in browser and terminal
- Include note about `/doc:refresh` endpoint for refreshing documentation cache

**FR-2.3: Maintain README.md Focus**
- Keep `README.md` focused on project overview and features only (per AGENTS.md)
- Preserve existing sections: Features, Quick Start, Architecture, API Pattern, Configuration, Contributing, License
- Ensure the file remains lean and serves as an entry point, not a comprehensive guide

#### FR-3: Update References in Other Documentation

**FR-3.1: Update INSTALL.md**
- Replace any references to `USAGE.md` with references to `/doc/` endpoint
- Update "Next Steps" section if it references `USAGE.md`

**FR-3.2: Update SPEC.md**
- Replace any references to `USAGE.md` with references to the documentation endpoints
- Ensure consistency in documentation references

**FR-3.3: Update AGENTS.md (if needed)**
- Replace any references to `USAGE.md` with references to `/doc/` or `/doc/md`
- Update any documentation workflow instructions

---

### Technical Requirements

#### TR-1: No Code Changes Required

**TR-1.1: Documentation Only**
- This PRD requires only documentation file changes
- No Go code modifications needed
- No API endpoint changes required
- No configuration changes required

#### TR-2: Documentation Template Verification

**TR-2.1: Completeness Check**
- Verify `cmd/moon/internal/handlers/templates/doc.md.tmpl` contains all essential information from `USAGE.md`
- Ensure template includes:
  - API endpoint documentation for all actions
  - Query parameter documentation (filtering, sorting, pagination, field selection, search)
  - Request/response examples with proper formatting
  - Authentication documentation
  - Error handling documentation
  - ULID format and benefits
  - URL prefix configuration guidance
  - Troubleshooting tips

**TR-2.2: Template Structure**
- Maintain existing template structure and formatting
- Ensure generated documentation is comprehensive and easy to navigate
- Keep Markdown formatting clean and consistent

#### TR-3: Backward Compatibility

**TR-3.1: No Breaking Changes**
- Removing `USAGE.md` does not affect the API or runtime behavior
- All API endpoints remain unchanged
- All configuration options remain unchanged
- Users accessing `/doc/` or `/doc/md` endpoints are unaffected

**TR-3.2: User Migration Path**
- Users currently referencing `USAGE.md` should be directed to `/doc/` or `/doc/md`
- Update GitHub wiki or external documentation to reference the new documentation location
- Consider adding a brief migration note in the next release notes

---

## API Specifications

### No API Changes

This PRD involves documentation changes only. No API endpoints, request/response formats, or behaviors are modified.

---

## Validation Rules and Constraints

### VR-1: Documentation Completeness

**VR-1.1: Content Coverage**
- All topics covered in `USAGE.md` must be present in `/doc/` and `/doc/md` output
- No information loss during migration
- Documentation must remain comprehensive and actionable

**VR-1.2: Example Coverage**
- All curl examples from `USAGE.md` must be present in generated documentation
- Examples must be copy-pasteable and functional
- Code blocks must have copy buttons (per PRD-039)

### VR-2: Documentation Quality

**VR-2.1: Clarity and Readability**
- Generated documentation must be clear, concise, and well-structured
- Markdown formatting must be correct and consistent
- Navigation must be intuitive

**VR-2.2: Accuracy**
- All documentation must reflect current API behavior
- Examples must use correct endpoint patterns and response formats
- Error messages and codes must match actual implementation

### VR-3: README.md Guidelines

**VR-3.1: Brevity**
- `README.md` should remain concise and focused
- Avoid duplicating detailed API documentation
- Provide clear pointers to `/doc/` for detailed information

**VR-3.2: Entry Point**
- `README.md` serves as the entry point for new users
- Must include enough information to get started quickly
- Should link to installation, documentation, and support resources

---

## Error Handling

### No Error Handling Changes

This PRD involves documentation changes only. No error handling logic is modified.

---

## Acceptance Criteria

### AC-1: USAGE.md Removal

**Verification:**
- [ ] `USAGE.md` file is deleted from repository root
- [ ] No references to `USAGE.md` exist in any documentation files
- [ ] No references to `USAGE.md` exist in any code comments
- [ ] Git history preserves the file for reference

**Test Cases:**
```bash
# Verify USAGE.md is removed
test ! -f USAGE.md

# Search for references
grep -r "USAGE.md" . --exclude-dir=.git
# Should return no results
```

### AC-2: README.md Updates

**Verification:**
- [ ] "Documentation & Support" section updated to reference `/doc/` and `/doc/md`
- [ ] Clear instructions for accessing documentation in browser and terminal
- [ ] Example curl commands for viewing documentation
- [ ] Note about `/doc:refresh` endpoint included
- [ ] No references to `USAGE.md` remain
- [ ] File remains concise and focused on project overview

**Expected Content:**
```markdown
## Documentation

Moon provides comprehensive, auto-generated API documentation:

- **HTML Documentation**: Visit `http://localhost:6006/doc/` in your browser for a complete, interactive API reference
- **Markdown Documentation**: Access `http://localhost:6006/doc/md` for terminal-friendly or AI-agent documentation
- **Refresh Documentation**: POST to `http://localhost:6006/doc:refresh` to update the documentation cache after schema changes

### Quick Access Examples

```bash
# View in browser
open http://localhost:6006/doc/

# View in terminal
curl http://localhost:6006/doc/md | less

# Refresh documentation cache
curl -X POST http://localhost:6006/doc:refresh
```

### Additional Resources

- [INSTALL.md](INSTALL.md): Installation and deployment guide
- [SPEC.md](SPEC.md): Architecture and technical specifications
- [samples/](samples/): Sample configuration files
- [scripts/](scripts/): Test and demo scripts
```

### AC-3: INSTALL.md Updates

**Verification:**
- [ ] All references to `USAGE.md` replaced with `/doc/` references
- [ ] "Next Steps" section updated appropriately
- [ ] Installation guide references the documentation endpoints

**Test Cases:**
```bash
# Check INSTALL.md for USAGE.md references
grep "USAGE.md" INSTALL.md
# Should return no results
```

### AC-4: SPEC.md Updates

**Verification:**
- [ ] All references to `USAGE.md` replaced with documentation endpoint references
- [ ] Documentation architecture section (if present) updated
- [ ] Consistency with actual implementation

### AC-5: Documentation Template Verification

**Verification:**
- [ ] Template at `cmd/moon/internal/handlers/templates/doc.md.tmpl` contains all essential content
- [ ] Generated documentation includes all API endpoints
- [ ] Generated documentation includes query parameters, examples, and error handling
- [ ] Generated documentation includes ULID format information
- [ ] Generated documentation includes authentication guidance
- [ ] Generated documentation includes troubleshooting tips

**Test Cases:**
```bash
# Start server
./moon --config samples/moon.conf

# Fetch generated markdown documentation
curl -s http://localhost:6006/doc/md > /tmp/generated-docs.md

# Verify completeness (manual review)
# - Check for collections:list, collections:create, etc.
# - Check for {collection}:list, {collection}:create, etc.
# - Check for filtering examples (price[gt]=100)
# - Check for aggregation endpoints (:count, :sum, :avg, etc.)
# - Check for ULID documentation
# - Check for authentication section
```

### AC-6: No Broken Links

**Verification:**
- [ ] All internal documentation links are valid
- [ ] No broken references to `USAGE.md`
- [ ] All references to documentation endpoints are correct

**Test Cases:**
```bash
# Check for broken markdown links in README.md
# (Manual or automated link checker)

# Verify documentation endpoints are accessible
curl -I http://localhost:6006/doc/
curl -I http://localhost:6006/doc/md
```

### AC-7: No Regression

**Verification:**
- [ ] All existing tests pass (no code changes expected)
- [ ] Documentation endpoints continue to work correctly
- [ ] No changes to API behavior or responses
- [ ] No changes to configuration options

**Test Cases:**
```bash
# Run all tests
go test ./... -v

# Run test scripts
./scripts/health.sh
./scripts/collection.sh
./scripts/data.sh
```

### AC-8: Quality and Consistency

**Verification:**
- [ ] All documentation follows consistent Markdown formatting
- [ ] Code examples are properly formatted with syntax highlighting
- [ ] All documentation is spell-checked and grammatically correct
- [ ] Documentation tone is professional and clear

---

## Implementation Checklist

- [x] Verify `cmd/moon/internal/handlers/templates/doc.md.tmpl` contains all essential content from `USAGE.md`
- [x] Delete `USAGE.md` from repository root
- [x] Update `README.md` to reference `/doc/` and `/doc/md` endpoints
- [x] Remove USAGE.md references from `INSTALL.md` (none existed)
- [x] Remove USAGE.md references from `SPEC.md` (none existed)
- [x] Remove USAGE.md references from `AGENTS.md` (if any)
- [x] Search codebase for any remaining `USAGE.md` references
- [x] Test documentation endpoints to ensure completeness
- [x] Run all tests and ensure 100% pass rate
- [x] Review all documentation for clarity, accuracy, and consistency
- [x] Update any external references (samples/README.md) to point to `/doc/`
- [x] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [x] Ensure all test scripts in `scripts/*.sh` are working properly and up to date with the latest code and API changes.

---

## Related PRDs

- [PRD-036: Documentation Endpoints](036-doc-endpoints.md) - Initial `/doc/` and `/doc/md` endpoint implementation
- [PRD-037: Documentation Template Refactoring](037-doc-template-refactoring.md) - Template structure and organization
- [PRD-039: Copy Button for Code Blocks](039-copy-button-code-blocks.md) - Enhanced documentation usability
- [PRD-001: Project Structure](001-project-structure.md) - Overall project organization
- [PRD-016: Health Check Endpoint](016-health-check-endpoint.md) - Referenced in documentation examples
