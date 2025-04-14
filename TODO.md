# TODO

## Configuration Management Implementation

### Dependencies
- [x] **Add Viper and Validator dependencies:** 
  - **Action:** Run `go get github.com/spf13/viper github.com/go-playground/validator/v10` and ensure `go.mod` and `go.sum` are updated.
  - **Depends On:** None
  - **AC Ref:** PLAN.md Step 6.1

### Define Configuration Structure
- [x] **Create Config struct in `config.go`:** 
  - **Action:** Create `internal/config/config.go`. Define the hierarchical `Config` struct with nested structs for `ServerConfig`, `DatabaseConfig`, `AuthConfig`, and `LLMConfig`. Add `mapstructure` and `validate:"required"` tags to each nested struct field.
  - **Depends On:** None
  - **AC Ref:** PLAN.md Step 3.1.1, 3.1.2, 3.1.3

- [x] **Define nested configuration structs:** 
  - **Action:** In `config.go`, implement the following structs with appropriate validation rules:
    1. `ServerConfig` with fields for `Port` (int) and `LogLevel` (string)
    2. `DatabaseConfig` with field for `URL` (string)
    3. `AuthConfig` with field for `JWTSecret` (string)
    4. `LLMConfig` with field for `GeminiAPIKey` (string)
    Add appropriate `mapstructure` tags and `validate` tags to each field as specified in PLAN.md.
  - **Depends On:** Create Config struct in `config.go`
  - **AC Ref:** PLAN.md Step 3.1.3, 3.1.4

### Implement Configuration Loading
- [x] **Create `load.go` with `Load()` function:** 
  - **Action:** Create `internal/config/load.go` with a `Load() (*Config, error)` function. Import necessary packages (`fmt`, `strings`, `github.com/go-playground/validator/v10`, `github.com/spf13/viper`). Initialize a new Viper instance.
  - **Depends On:** Add Viper and Validator dependencies, Create Config struct in `config.go`
  - **AC Ref:** PLAN.md Step 3.2.1

- [ ] **Implement configuration loading logic:** 
  - **Action:** Within `Load()`, implement the following in sequence:
    1. Set default values for non-critical parameters (e.g., port, log level)
    2. Configure Viper to look for config files (e.g., `config.yaml`) in the working directory
    3. Configure Viper to bind environment variables with "SCRY_" prefix
    4. Unmarshal the loaded configuration into the `Config` struct
    5. Validate the config using validator and return meaningful errors
  - **Depends On:** Create `load.go` with `Load()` function
  - **AC Ref:** PLAN.md Step 3.2.2, 3.2.3, 3.2.4

### Integration in Main Application
- [ ] **Update `main.go` to use configuration:** 
  - **Action:** Update `cmd/server/main.go` to import the `internal/config` package. Call `config.Load()` early in the `main` function. Handle errors by logging and terminating if loading fails. Add a simple log message showing loaded values and a placeholder comment for future dependency injection.
  - **Depends On:** Implement configuration loading logic
  - **AC Ref:** PLAN.md Step 3.3.1, 3.3.2, 3.3.3

### Supporting Files
- [ ] **Create example configuration files:** 
  - **Action:** Create two files:
    1. `.env.example` containing documented variables with the "SCRY_" prefix
    2. `config.yaml.example` with the same variables in YAML format
    Both files should include examples for server, database, auth, and LLM configuration.
  - **Depends On:** Create Config struct in `config.go`
  - **AC Ref:** PLAN.md Step 3.4.1, 3.4.3

- [x] **Update `.gitignore` to exclude `.env`:** 
  - **Action:** Add `.env` to the project's `.gitignore` file to prevent accidental commits of local environment settings.
  - **Depends On:** None
  - **AC Ref:** PLAN.md Step 3.4.2

### Documentation
- [ ] **Add GoDoc comments:** 
  - **Action:** Add comprehensive GoDoc comments to:
    1. The `Config` struct and all nested structs in `config.go`
    2. The `Load()` function in `load.go`, explaining precedence rules
    3. Update or create `doc.go` for package-level documentation
  - **Depends On:** Create Config struct in `config.go`, Implement configuration loading logic
  - **AC Ref:** PLAN.md Step 3.5.2

- [ ] **Update `README.md` with configuration details:** 
  - **Action:** Update `README.md` with:
    1. Required environment variables (referencing `.env.example`)
    2. Optional config file usage and location
    3. Example local setup instructions
    4. Ensure the `config` package is described in Architecture Overview
  - **Depends On:** Create example configuration files
  - **AC Ref:** PLAN.md Step 3.5.1

### Testing
- [ ] **Write unit tests for `Load()` function:** 
  - **Action:** Create `internal/config/load_test.go` with test cases covering:
    1. Loading with environment variables set
    2. Loading with a config file
    3. Testing precedence (env vars over file)
    4. Testing validation with missing required values
    5. Testing validation with invalid values
  - **Depends On:** Implement configuration loading logic
  - **AC Ref:** PLAN.md Step 4.1

- [ ] **Create basic integration test:** 
  - **Action:** Create a basic integration test that initializes the application with configuration to verify proper loading and injection of the configuration. This may be in `cmd/server/main_test.go` or similar location.
  - **Depends On:** Update `main.go` to use configuration
  - **AC Ref:** PLAN.md Step 4.2

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
- [ ] **Issue/Assumption:** Only implementing initially required fields specified in PLAN.md examples.
  - **Context:** PLAN.md includes comments like "Add other server settings as needed". This implementation covers only the explicitly mentioned fields (Port, LogLevel, URL, JWTSecret, GeminiAPIKey). Future fields will require extending the structs.

- [ ] **Issue/Assumption:** Dependency Injection mechanism is left as a placeholder.
  - **Context:** PLAN.md Step 3.3.3 requires passing config via dependency injection. The implementation adds only a placeholder for this, as the actual DI mechanism will likely depend on future architectural decisions.