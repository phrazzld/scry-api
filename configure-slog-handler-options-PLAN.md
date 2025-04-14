# Configure Slog Handler Options

## Implementation Approach
Create an `slog.HandlerOptions` struct within the `Setup` function after the log level is determined. Set the `Level` field to the parsed log level. Add a commented-out `AddSource` field to indicate it's intentionally disabled for performance reasons.
