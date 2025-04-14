# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands
- Go: `go build ./cmd/server` - Build the server binary
- Test: `go test ./...` - Run all tests
- Single test: `go test -v ./path/to/package -run TestName` 
- Test with coverage: `go test -cover ./...`

## Code Style
- Go fmt: `gofmt -w .` or `goimports -w .` - Format code
- Linting: `golangci-lint run` - Run linters with strict settings
- Types: Use strict typing - never use `interface{}` without necessity
- Naming: `camelCase` for unexported, `PascalCase` for exported
- Error handling: Always handle errors explicitly, no empty catch blocks
- Immutability: Prefer immutable data structures and pure functions
- Core principles: Simplicity, Maintainability, Explicit over Implicit
- Comments: Explain WHY, not WHAT (code should be self-explanatory)
- Testing: Follow FIRST principles (Fast, Independent, Repeatable, Self-validating, Timely)