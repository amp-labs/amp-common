# Package: cmd

Fluent API for building and executing shell commands with enhanced error handling.

## Usage

```go
import "github.com/amp-labs/amp-common/cmd"

// Basic command execution
exitCode, err := cmd.New(ctx, "git", "status").
    SetDir("/path/to/repo").
    SetStdout(os.Stdout).
    Run()

// Capture output with observers
var output []byte
exitCode, err := cmd.New(ctx, "ls", "-la").
    SetStdoutObserver(func(b []byte) { output = b }).
    Run()

// Environment variables
exitCode, err := cmd.New(ctx, "env").
    AppendEnv("MY_VAR=value").
    Run()
```

## Common Patterns

- Fluent chainable methods: SetDir, SetStdin, SetStdout, SetStderr
- Output observers for capturing results
- Environment variable management (AppendEnv, PrependEnv, SetEnv)
- Returns both exit code and error separately
- Inherits current process environment

## Gotchas

- Returns exit code + error (not just error)
- Observers called after command finishes
- Context cancellation supported
