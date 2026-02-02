# Package: script

Utilities for running scripts with standardized logging, signal handling, and exit codes.

## Usage

```go
func main() {
    script.Run(func(ctx context.Context) error {
        // Your script logic here
        if someCondition {
            return script.Exit(0)  // Exit with code
        }
        return script.ExitWithError(err)  // Exit with error
    }, script.LogLevel(slog.LevelDebug))
}
```

## Common Patterns

- `Run()` - Execute script with automatic setup
- `Exit(code)` - Exit with specific code
- `ExitWithError(err)` - Exit with code 1 and log error
- Options: LogLevel, LogOutput, EnableFlagParse
- Automatic signal handling (SIGINT, SIGTERM)

## Gotchas

- Sets up logger, flag parsing, and signal handlers automatically
- Use Exit() family of functions for controlled script termination
- Build info displayed if available

## Related

- `logger` - Logging setup
- `shutdown` - Signal handling
