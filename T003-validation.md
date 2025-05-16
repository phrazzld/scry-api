# T003 Validation Criteria

## Pre-Implementation Checks
1. [x] Identify all files containing `go run cmd/server/main.go`
2. [x] Document current command patterns across all files
3. [x] List all documentation files that need updates

## Post-Implementation Validation

### 1. Command Pattern Validation
- [x] No instances of `go run cmd/server/main.go` remain in any `.md` file (except explanatory context)
- [x] All Go commands use package-based execution (`./cmd/server`)
- [x] Makefile targets are used in documentation instead of raw commands

### 2. Makefile Functionality
- [x] `make run-server` executes successfully
- [x] `make migrate-up` applies migrations correctly
- [x] `make migrate-down` rolls back migrations
- [x] `make build` creates binary successfully
- [x] `make test` runs all tests
- [x] `make test-integration` runs integration tests
- [x] `make lint` runs linter
- [x] `make fmt` formats code

### 3. Documentation Completeness
- [x] docs/DEVELOPMENT_GUIDE.md exists and contains:
  - [x] All Makefile targets documented
  - [x] Usage examples for each target
  - [x] Flag/argument documentation
  - [x] Troubleshooting section
- [x] README.md updated with:
  - [x] Makefile commands in Getting Started
  - [x] Link to DEVELOPMENT_GUIDE.md
  - [x] No raw Go commands
- [x] CLAUDE.md updated with:
  - [x] Makefile commands
  - [x] Reference to DEVELOPMENT_GUIDE.md

### 4. Cross-References
- [x] All documentation files reference DEVELOPMENT_GUIDE.md appropriately
- [x] No conflicting command documentation
- [x] Consistent terminology throughout

### 5. User Experience
- [x] New developer can follow README.md to set up project
- [x] Commands are discoverable and well-explained
- [x] No ambiguity about which commands to use
