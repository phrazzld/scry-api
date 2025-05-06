# PostgreSQL in GitHub Actions for Test Coverage

## Task Overview

Brainstorm the best designs, architectures, and approaches for adding PostgreSQL to GitHub Actions for our Go project. The goal is to ensure proper test coverage both in general development and in CI environments.

## Current Situation

1. We have a Go API (Scry API) for spaced repetition flashcards with PostgreSQL database dependency
2. Integration tests are currently skipped in CI due to no database connection
3. CI fails with exit code 1 because all tests are skipped and no tests actually run
4. We want to properly test our database interactions in CI

## Specific Requirements

1. **Brainstorm multiple approaches** for adding PostgreSQL to GitHub Actions workflows
   - Consider container-based solutions
   - Consider managed/hosted options
   - Consider test doubles/mocks if appropriate
   - Consider any alternative approaches

2. **Evaluate tradeoffs** for each approach:
   - Setup complexity
   - Maintenance burden
   - Test fidelity (how close to production)
   - Speed/performance in CI
   - Resource usage/costs

3. **Make a strong recommendation** for the best approach based on:
   - Alignment with our development philosophy (testability, simplicity, explicitness)
   - Practical considerations for CI environment
   - Best practices in the Go ecosystem

4. **Create a highly detailed implementation plan** that includes:
   - GitHub Actions workflow changes
   - Required environment variables and secrets
   - Database initialization/migration approach
   - Test configurations/modifications
   - Any local development changes needed

5. **Break down the plan into atomic tasks** formatted as a TODO.md file with:
   - Checkbox items starting with `- [ ]`
   - Well-defined, narrowly scoped, actionable tasks
   - Clear success criteria for each task
   - Logical grouping and sequencing

## Project Context

- Go project with PostgreSQL database
- Uses transaction isolation in tests
- Has existing migration system
- Integration tests skip when no database connection available
- Goal is for CI to run ALL tests including database tests
