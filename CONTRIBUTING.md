# Contributing to Scry API

Thank you for your interest in contributing to Scry API! This document provides guidelines and instructions for contributors.

## Development Setup

### Prerequisites
- Go 1.23+
- PostgreSQL with `pgvector` extension (for production, or via Docker for development)
- Gemini API key for LLM integration

### Setting Up Your Development Environment

1. **Clone the repository**:
   ```bash
   git clone https://github.com/phrazzld/scry-api.git
   cd scry-api
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Set up pre-commit hooks** (required for all contributors):
   ```bash
   # Install pre-commit if you don't have it
   # macOS
   brew install pre-commit

   # Python/pip (any platform)
   pip install pre-commit

   # Install the hooks
   pre-commit install --install-hooks

   # Install the pre-push hook specifically
   pre-commit install --hook-type pre-push
   ```

4. **Configure environment**:
   - Copy `config.yaml.example` to `config.yaml` and update with your local settings
   - Alternatively, set up environment variables as described in README.md

5. **Start a local database**:
   ```bash
   docker-compose -f infrastructure/local_dev/docker-compose.yml up -d
   ```

## Development Workflow

### Git Workflow

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**, following our [Development Philosophy](docs/DEVELOPMENT_PHILOSOPHY.md)

3. **Commit your changes** using [Conventional Commits](https://www.conventionalcommits.org/) format:
   ```bash
   git commit -m "feat: add new user setting feature"
   ```

4. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

   **Note**: Before pushing, a pre-push hook will run the entire test suite. Your code must pass all tests before it can be pushed to the remote repository. This ensures that no code with failing tests is ever pushed to the shared repository.

5. **Create a pull request** against the `master` branch

### Code Style and Standards

- Follow the guidelines in [DEVELOPMENT_PHILOSOPHY.md](docs/DEVELOPMENT_PHILOSOPHY.md)
- All code must pass the pre-commit hooks and CI checks
- Write comprehensive tests for new functionality

### Pre-commit Hooks

We use pre-commit hooks to ensure code quality:

- **Pre-commit Hooks**: Run before each commit to ensure code formatting, linting, and basic correctness
- **Pre-push Hooks**: Run before pushing to remote to ensure all tests pass

#### Available Hooks

**Pre-commit Hooks**:
- Code formatting (gofmt/goimports)
- Linting (golangci-lint)
- Build verification
- Basic validity checks

**Pre-push Hooks**:
- Full test suite execution with `-tags=test_without_external_deps`

#### Bypassing Hooks (Emergency Only)

In exceptional circumstances, hooks can be bypassed:

```bash
# Skip pre-commit hooks (NOT RECOMMENDED)
git commit --no-verify -m "feat: your feature"

# Skip pre-push hooks (NOT RECOMMENDED)
git push --no-verify origin your-branch
```

**WARNING**: Bypassing hooks should be done only in exceptional circumstances and with thorough justification. The team should be informed of any bypassed hooks and the code should be verified manually.

## Testing

- Run tests with appropriate tags:
  ```bash
  # Run all tests using mock implementations
  go test -v -tags=test_without_external_deps ./...

  # Run integration tests (requires external dependencies)
  go test -v -tags=integration ./...
  ```

- See [DEVELOPMENT_PHILOSOPHY.md](docs/DEVELOPMENT_PHILOSOPHY.md#testing-strategy) for our testing strategy and guidelines

## Pull Request Process

1. Ensure all tests pass locally before submitting
2. Update documentation if necessary
3. Fill out the pull request template completely
4. Request review from at least one team member
5. Address all review comments

## Additional Resources

- [Development Guide](docs/DEVELOPMENT_GUIDE.md)
- [Architecture Guidelines](docs/philosophy/ARCHITECTURE_GUIDELINES.md)
- [Code Review Checklist](docs/CODE_REVIEW_CHECKLIST.md)
