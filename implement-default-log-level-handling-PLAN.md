# Implement Default Log Level Handling

## Implementation Approach
Add a default case to the log level switch statement that sets the level to `slog.LevelInfo` when an invalid log level is configured. Create a temporary logger (using `slog.New` with a basic text handler) to log a warning that indicates the invalid level that was configured and that the default (info) level is being used instead.
