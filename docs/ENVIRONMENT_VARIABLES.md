# Environment Variable Reference

This document serves as the authoritative reference for all environment variables used in the Scry API project. It defines standardized naming conventions, precedence rules, default values, and provides a comprehensive list of all supported variables.

## Naming Convention Standard

To ensure consistency and clarity, Scry API follows these environment variable naming conventions:

1. **Application Configuration Variables**
   - Format: `SCRY_<MODULE>_<SETTING>` (uppercase with underscores)
   - Example: `SCRY_SERVER_PORT`
   - Purpose: Control application behavior and settings

2. **Testing/CI-Specific Variables**
   - Format: `SCRY_TEST_<PURPOSE>` for test-specific settings
   - Example: `SCRY_TEST_DB_URL`
   - Purpose: Configure test environments and CI pipeline

3. **Standard/External Variables** (not changeable)
   - Format: Keep standard names as-is
   - Example: `DATABASE_URL`, `CI`, `GITHUB_ACTIONS`
   - Purpose: Maintain compatibility with standard tools and platforms
   
## Precedence Rules

Environment variables are loaded in the following order of precedence:

1. **Environment Variables** (highest priority)
   - Variables set in the shell environment override all other sources
   - Standardized `SCRY_` prefixed variables take precedence over legacy variables

2. **Configuration File** (middle priority)
   - Values from `config.yaml` file are used if no environment variable is set
   - File must be in the working directory where the application is run

3. **Default Values** (lowest priority)
   - Hardcoded defaults in `internal/config/load.go` are used as a last resort
   - Not all settings have default values; some are required

## Configuration Variable Reference

### Server Configuration

| Environment Variable | Config Path | Type | Default | Required | Description |
|---------------------|-------------|------|---------|----------|-------------|
| `SCRY_SERVER_PORT` | `Config.Server.Port` | int | 8080 | Yes | HTTP server port number |
| `SCRY_SERVER_LOG_LEVEL` | `Config.Server.LogLevel` | string | "info" | Yes | Log level (debug, info, warn, error) |

### Database Configuration

| Environment Variable | Config Path | Type | Default | Required | Description |
|---------------------|-------------|------|---------|----------|-------------|
| `SCRY_DATABASE_URL` | `Config.Database.URL` | string | - | Yes | Primary database connection string |
| `SCRY_TEST_DB_URL` | - | string | - | No | Test-specific database connection (not in Config struct) |
| `DATABASE_URL` | - | string | - | No | Standard database URL (legacy support, not in Config struct) |

### Authentication Configuration

| Environment Variable | Config Path | Type | Default | Required | Description |
|---------------------|-------------|------|---------|----------|-------------|
| `SCRY_AUTH_JWT_SECRET` | `Config.Auth.JWTSecret` | string | - | Yes | Secret key for JWT signing (32+ chars) |
| `SCRY_AUTH_BCRYPT_COST` | `Config.Auth.BCryptCost` | int | 10 | No | Bcrypt hashing cost (4-31) |
| `SCRY_AUTH_TOKEN_LIFETIME_MINUTES` | `Config.Auth.TokenLifetimeMinutes` | int | 60 | Yes | JWT access token lifetime in minutes |
| `SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES` | `Config.Auth.RefreshTokenLifetimeMinutes` | int | 10080 (7 days) | Yes | JWT refresh token lifetime in minutes |

### Language Model Configuration

| Environment Variable | Config Path | Type | Default | Required | Description |
|---------------------|-------------|------|---------|----------|-------------|
| `SCRY_LLM_GEMINI_API_KEY` | `Config.LLM.GeminiAPIKey` | string | - | Yes | Google Gemini API key |
| `SCRY_LLM_MODEL_NAME` | `Config.LLM.ModelName` | string | "gemini-2.0-flash" | Yes | Gemini model name |
| `SCRY_LLM_PROMPT_TEMPLATE_PATH` | `Config.LLM.PromptTemplatePath` | string | - | Yes | Path to prompt template file |
| `SCRY_LLM_MAX_RETRIES` | `Config.LLM.MaxRetries` | int | 3 | No | Maximum number of API retries |
| `SCRY_LLM_RETRY_DELAY_SECONDS` | `Config.LLM.RetryDelaySeconds` | int | 2 | No | Base delay between retries (seconds) |

### Task Configuration

| Environment Variable | Config Path | Type | Default | Required | Description |
|---------------------|-------------|------|---------|----------|-------------|
| `SCRY_TASK_WORKER_COUNT` | `Config.Task.WorkerCount` | int | 2 | Yes | Number of worker goroutines |
| `SCRY_TASK_QUEUE_SIZE` | `Config.Task.QueueSize` | int | 100 | Yes | Size of in-memory task queue |
| `SCRY_TASK_STUCK_TASK_AGE_MINUTES` | `Config.Task.StuckTaskAgeMinutes` | int | 30 | Yes | Minutes before a task is considered stuck |

## CI and Testing Variables

These variables are primarily used by the CI system and test framework:

| Environment Variable | Default | Required | Description |
|---------------------|---------|----------|-------------|
| `SCRY_PROJECT_ROOT` | - | No | Explicit override for project root directory |
| `CI` | - | No | Set to any value to indicate CI environment |
| `GITHUB_ACTIONS` | - | No | Set to any value to indicate GitHub Actions CI |
| `GITHUB_WORKSPACE` | - | No | Root directory of GitHub repository |
| `GITLAB_CI` | - | No | Set to any value to indicate GitLab CI |
| `CI_PROJECT_DIR` | - | No | Root directory of GitLab repository |

## Environment-Specific Configurations

### Local Development

For local development, create a `config.yaml` file in the project root with the necessary settings:

```yaml
server:
  port: 8080
  log_level: debug

database:
  url: postgres://localhost:5432/scry_dev?sslmode=disable

auth:
  jwt_secret: your-secret-key-at-least-32-characters-long
  bcrypt_cost: 10

llm:
  gemini_api_key: your-gemini-api-key
  model_name: gemini-2.0-flash
  prompt_template_path: prompts/flashcard_template.txt

task:
  worker_count: 2
  queue_size: 100
  stuck_task_age_minutes: 30
```

You can also set any of these values as environment variables with the `SCRY_` prefix.

### CI Environment

In CI environments, the system enforces certain standards, particularly for database connections:

- Username and password standardized to `postgres:postgres`
- Standard URL format: `postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable`
- Automatic correction of non-compliant credentials
- Synchronization of all database URL environment variables

See [CI Environment Configuration](./ci_environment.md) for detailed information about CI-specific settings.

## Backward Compatibility

For a transitional period, the system supports legacy environment variable names alongside standardized names. Whenever a legacy variable is detected, a warning will be logged, and developers should update to the standardized naming convention.

Legacy variables include:
- `DATABASE_URL` (use `SCRY_DATABASE_URL` instead)
- Direct database URL environment variables used outside the configuration system