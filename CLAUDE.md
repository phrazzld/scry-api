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

## Project-Specific Tools

### Architect CLI
- IMPORTANT: The project has an `architect` CLI tool available for invoking on-demand intelligence with project-specific context.
- Use `architect` for a wide range of needs, not just implementation plans:
  - Generate implementation approaches for complex problems
  - Answer difficult technical questions about the codebase
  - Brainstorm ideas and solutions
  - Analyze tradeoffs between different approaches
  - Debug complex issues
  - Get insights on architectural decisions
- Use it when you encounter challenges, struggles, or complex tasks that benefit from additional context-aware analysis.
- Always include `docs/DEVELOPMENT_PHILOSOPHY.md` as context at a minimum.
- You can (and should) include many relevant files as context - the tool can handle a significant amount of context.
- Basic usage: `architect --instructions <instructions-file.md> --output-dir architect_output <context-files>`
- Check all available options with `architect --help`
- Example:
  ```
  architect --instructions questions.md --output-dir architect_output docs/DEVELOPMENT_PHILOSOPHY.md path/to/relevant-file1.go path/to/relevant-file2.go
  ```
- Review the generated output in the architect_output directory for insights before proceeding.
