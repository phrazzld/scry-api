# Code Review Checklist

This checklist serves as a guide for code reviewers to ensure consistency, quality, and adherence to project standards. Not all items will apply to every pull request.

## Core Requirements

- [ ] Code follows [Development Philosophy](./DEVELOPMENT_PHILOSOPHY.md) and language-specific appendices
- [ ] Changes include appropriate tests (unit and/or integration)
- [ ] All tests pass locally and in CI
- [ ] Code is properly formatted (`make fmt`)
- [ ] Linting passes without errors (`make lint`)
- [ ] Commit messages follow Conventional Commits specification
- [ ] No hardcoded secrets or sensitive information

## Go-Specific Checks

- [ ] Verify database driver imports in files using `sql.Open()` (e.g., `_ "github.com/jackc/pgx/v5/stdlib"`)
- [ ] Check for restrictive build tags on core application files (ensure main package files don't have tags that prevent compilation)
- [ ] Error handling follows Go conventions (explicit checks, proper wrapping)
- [ ] No unnecessary mocking of internal collaborators
- [ ] Context propagation is maintained through the call chain
- [ ] Goroutines and channels are used only when necessary
- [ ] No `panic()` for normal error conditions

## Database and Migrations

- [ ] Migration files follow naming conventions and are idempotent
- [ ] Database queries include proper error handling
- [ ] SQL queries have deterministic ordering (ORDER BY with secondary sort key)
- [ ] Connection pools are properly configured and closed

## Security

- [ ] Input validation at all system boundaries
- [ ] Authentication and authorization checks are in place
- [ ] No SQL injection vulnerabilities (use parameterized queries)
- [ ] Sensitive data is properly redacted in logs
- [ ] Dependencies are up-to-date and free from known vulnerabilities

## Performance

- [ ] No obvious performance bottlenecks introduced
- [ ] Database queries are optimized (appropriate indexes, avoid N+1)
- [ ] Large data operations are paginated
- [ ] Resource cleanup is properly handled (defer close, etc.)

## Documentation

- [ ] Code changes include necessary documentation updates
- [ ] Public APIs have appropriate doc comments
- [ ] Complex logic includes explanatory comments focusing on "why"
- [ ] README or relevant documentation is updated if behavior changes

## Testing

- [ ] Test coverage meets or exceeds project thresholds
- [ ] Tests are deterministic and don't rely on external state
- [ ] Integration tests use proper test isolation patterns
- [ ] No skipped tests without valid justification

## CI/CD

- [ ] Changes don't break the CI pipeline
- [ ] Pre-commit hooks are properly configured and pass
- [ ] Build tags are used appropriately for test isolation
- [ ] Deployment considerations are addressed (if applicable)

## Final Checks

- [ ] Code doesn't introduce circular dependencies
- [ ] File sizes are reasonable (< 1000 lines)
- [ ] No commented-out code left in the codebase
- [ ] Changes are focused and don't include unrelated modifications
