# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

The project uses a Makefile for standardized commands. See all available commands with `make help`.

- **Build:** `make build`
- **Run server:** `make run-server`
- **Format code:** `make fmt`
- **Lint code:** `make lint`
- **Run all tests:** `make test`
- **Run specific test:** `go test -v ./path/to/package -run TestName`
- **Run tests without external deps:** `make test-no-deps`
- **Database migrations:** `make migrate-up`

For comprehensive command documentation, see [docs/DEVELOPMENT_GUIDE.md](docs/DEVELOPMENT_GUIDE.md).

## Coding Standards

- **Formatting:** Code must be formatted with `gofmt`/`goimports`, 120 char line limit
- **Linting:** Use `golangci-lint` with project config, never suppress errors
- **Error handling:** Use explicit error checking, prefer `errors.Is`/`errors.As`, wrap with `fmt.Errorf("%w", err)`
- **Naming:** PascalCase for exported identifiers, camelCase for unexported
- **Package structure:** Package by feature, no utility packages, no circular dependencies
- **Testing:** Write table-driven tests, use transaction isolation pattern from testutils, no mocking internal collaborators
- **Context:** Always propagate context.Context, include correlation_id
- **Logging:** Use structured logging with log/slog

## Core Principles

- **Simplicity First:** Minimize complexity, favor explicit over implicit
- **Design for Testability:** Write testable code, refactor when testing is difficult
- **Document Why, Not How:** Code should be self-documenting
- **TDD Approach:** Write tests before implementation when possible
- **Conventional Commits:** Follow specification for version control, always write detailed multiline commit messages
- **NEVER sign your commit messages. Your commit messages should ALWAYS and ONLY be detailed multiline conventional commits**

Refer to DEVELOPMENT_PHILOSOPHY.md for complete guidelines.
