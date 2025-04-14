# TODO

## Logger Package Setup
- [x] **Create Logger Package Directory Structure:**
  - **Action:** Create the directory `internal/platform/logger`.
  - **Depends On:** None
  - **AC Ref:** Structured logging framework implementation

- [x] **Create Initial Logger Go File:**
  - **Action:** Create the file `internal/platform/logger/logger.go`.
  - **Depends On:** Create Logger Package Directory Structure
  - **AC Ref:** Structured logging framework implementation

## Logger Core Implementation
- [x] **Implement Logger Setup Function Signature:**
  - **Action:** Define the `Setup` function signature in `logger.go` accepting `config.ServerConfig` and returning `(*slog.Logger, error)`. Include necessary imports (`log/slog`, `os`, `strings`, `config`).
  - **Depends On:** Create Initial Logger Go File
  - **AC Ref:** Configure logging system using application configuration

- [x] **Implement Log Level Parsing Logic:**
  - **Action:** Implement the `switch` statement within `Setup` to parse `cfg.LogLevel` (case-insensitive) and map it to the corresponding `slog.Level` constant (Debug, Info, Warn, Error).
  - **Depends On:** Implement Logger Setup Function Signature
  - **AC Ref:** Configure logging system using application configuration

- [x] **Implement Default Log Level Handling:**
  - **Action:** Add the `default` case to the log level switch statement. Set the level to `slog.LevelInfo` and log a warning using a temporary `slog` logger indicating the invalid configured level and the default being used.
  - **Depends On:** Implement Log Level Parsing Logic
  - **AC Ref:** Configure logging system using application configuration

- [x] **Configure Slog Handler Options:**
  - **Action:** Create an `slog.HandlerOptions` struct within `Setup`, setting the `Level` field based on the parsed level. Keep `AddSource` commented out for now.
  - **Depends On:** Implement Default Log Level Handling
  - **AC Ref:** Set up basic structured logging framework

- [x] **Create Slog JSON Handler:**
  - **Action:** Create a `slog.NewJSONHandler` within `Setup`, passing `os.Stdout` and the configured handler options.
  - **Depends On:** Configure Slog Handler Options
  - **AC Ref:** Set up basic structured logging framework

- [x] **Create and Set Default Slog Logger:**
  - **Action:** Create the main `slog.Logger` instance using `slog.New` with the JSON handler. Set this logger as the application's default using `slog.SetDefault`.
  - **Depends On:** Create Slog JSON Handler
  - **AC Ref:** Set up basic structured logging framework

- [x] **Finalize Logger Setup Function:**
  - **Action:** Ensure the `Setup` function returns the created `*slog.Logger` instance and a `nil` error on success.
  - **Depends On:** Create and Set Default Slog Logger
  - **AC Ref:** Set up basic structured logging framework

## Integration into `main.go`
- [x] **Import Logger and Slog in `main.go`:**
  - **Action:** Add imports for `log/slog`, `os`, and the new `internal/platform/logger` package in `cmd/server/main.go`. Remove the standard `log` import if no longer used.
  - **Depends On:** Create Initial Logger Go File
  - **AC Ref:** Configure logging system using application configuration

- [x] **Integrate Logger Setup into `initializeApp`:**
  - **Action:** In `initializeApp`, after `config.Load()` succeeds, call `logger.Setup(cfg.Server)`. Handle any potential error returned from `logger.Setup` by returning a wrapped error.
  - **Depends On:** Finalize Logger Setup Function, Import Logger and Slog in `main.go`
  - **AC Ref:** Configure logging system using application configuration

- [x] **Replace `fmt.Printf` with Structured Logging in `initializeApp`:**
  - **Action:** Replace the `fmt.Printf` call logging configuration details in `initializeApp` with `slog.Info` and `slog.Debug` calls, logging relevant config fields as structured attributes.
  - **Depends On:** Integrate Logger Setup into `initializeApp`
  - **AC Ref:** Implement appropriate log levels

- [x] **Replace Startup/Error Logging with Slog in `main`:**
  - **Action:** In the `main` function, replace the initial `fmt.Println` with `slog.Info`. Replace `log.Fatalf` with `slog.Error` followed by `os.Exit(1)` for initialization errors. Add an `slog.Info` message upon successful initialization.
  - **Depends On:** Integrate Logger Setup into `initializeApp`
  - **AC Ref:** Implement appropriate log levels

## Logger Testing
- [x] **Create Logger Test File:**
  - **Action:** Create the test file `internal/platform/logger/logger_test.go`.
  - **Depends On:** Create Logger Package Directory Structure
  - **AC Ref:** Structured logging framework implementation

- [x] **Implement Test Setup for Output Capture:**
  - **Action:** In `logger_test.go`, set up tests to capture log output by using a `bytes.Buffer` and custom handler for verification. Configure the test environment to ensure isolation.
  - **Depends On:** Create Logger Test File
  - **AC Ref:** Structured logging framework implementation

- [x] **Write Test for Valid Log Level Parsing:**
  - **Action:** Create test cases in `logger_test.go` that call `Setup` with different valid log levels ("debug", "info", "warn", "error") and verify each returns a logger with the correct level.
  - **Depends On:** Finalize Logger Setup Function, Implement Test Setup for Output Capture
  - **AC Ref:** Configure logging system using application configuration

- [x] **Write Test for Invalid Log Level Parsing:**
  - **Action:** Create a test case that calls `Setup` with an invalid `LogLevel`. Verify the returned logger's level defaults to `slog.LevelInfo` and that a warning message is logged to the captured output.
  - **Depends On:** Finalize Logger Setup Function, Implement Test Setup for Output Capture
  - **AC Ref:** Configure logging system using application configuration

- [x] **Verify JSON Output Format in Tests:**
  - **Action:** Add tests to verify that log output is correctly formatted as JSON, ensuring the structured logging is working properly.
  - **Depends On:** Write Test for Valid Log Level Parsing
  - **AC Ref:** Set up basic structured logging framework

## Documentation Updates
- [x] **Verify/Update `config.yaml.example`:**
  - **Action:** Check the `config.yaml.example` file and ensure the `log_level` setting description accurately reflects the supported options (`debug`, `info`, `warn`, `error`) and the default (`info`).
  - **Depends On:** Finalize Logger Setup Function
  - **AC Ref:** Configure logging system using application configuration

- [x] **Update `README.md` Architecture Overview:**
  - **Action:** Add a sentence or bullet point to the Architecture Overview section in `README.md` mentioning the use of the standard library's `log/slog` for structured logging via the `internal/platform/logger` package.
  - **Depends On:** Finalize Logger Setup Function
  - **AC Ref:** Structured logging framework implementation

- [x] **Add Godoc Comments to Logger Package:**
  - **Action:** Write clear godoc comments for the `logger` package itself, the `Setup` function, explaining their purpose, parameters, return values, and behavior (including default log level handling).
  - **Depends On:** Finalize Logger Setup Function
  - **AC Ref:** Structured logging framework implementation

## Contextual Logging Helpers (Future Extension)
- [x] **Define `loggerKey` Type:**
  - **Action:** Define an unexported type `loggerKey struct{}` in `logger.go` to be used as a safe key for storing/retrieving the logger from `context.Context`.
  - **Depends On:** Finalize Logger Setup Function
  - **AC Ref:** Implement contextual logging helpers

- [x] **Implement `WithRequestID` Helper Function:**
  - **Action:** Implement the `WithRequestID` function as shown in the plan, which takes a context and request ID, creates a logger with the ID field, and returns a new context containing this logger.
  - **Depends On:** Define `loggerKey` Type
  - **AC Ref:** Implement contextual logging helpers

- [x] **Implement `FromContext` Helper Function:**
  - **Action:** Implement the `FromContext` function as shown in the plan, which retrieves a logger from the context using `loggerKey` or returns the default logger if none is found.
  - **Depends On:** Define `loggerKey` Type
  - **AC Ref:** Implement contextual logging helpers

- [x] **Implement `LogWithContext` Helper Function:**
  - **Action:** Implement the `LogWithContext` function as shown in the plan, which retrieves the logger from context using `FromContext` and logs the message with provided arguments.
  - **Depends On:** Implement `FromContext` Helper Function
  - **AC Ref:** Implement contextual logging helpers

- [x] **Add Tests for Contextual Logging Helpers:**
  - **Action:** Create test cases in `logger_test.go` to verify the functionality of `WithRequestID`, `FromContext`, and `LogWithContext`. Test scenarios with and without a logger present in the context.
  - **Depends On:** Implement `LogWithContext` Helper Function
  - **AC Ref:** Implement contextual logging helpers

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
- [ ] **Issue/Assumption:** `AddSource: true` in `slog.HandlerOptions` should be left commented out for now.
  - **Context:** `PLAN.md` > Implementation Steps > 2. Implement Setup Function shows this commented. Appears to be intentional to avoid performance overhead of source location tracking.

- [ ] **Issue/Assumption:** Contextual logging helpers implementation is part of the task but may be considered lower priority.
  - **Context:** `PLAN.md` section 3 "Contextual Logging Helpers" and "Future Considerations" suggest these are planned but potentially deferrable after core logging functionality is in place.

- [ ] **Issue/Assumption:** Warning for invalid log level should use a temporary logger.
  - **Context:** The implementation pattern in `PLAN.md` uses a direct call to `slog.Warn` before the default logger is set, which might result in output to the default handler rather than our JSON handler.
