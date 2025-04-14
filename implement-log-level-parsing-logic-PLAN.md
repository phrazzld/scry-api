# Implement Log Level Parsing Logic

## Implementation Approach
Implement a switch statement in the `Setup` function that takes the `cfg.LogLevel` string (with case-insensitive matching using `strings.ToLower()`) and maps it to the appropriate `slog.Level` constant (Debug, Info, Warn, Error). The default case will be added in a subsequent task.
