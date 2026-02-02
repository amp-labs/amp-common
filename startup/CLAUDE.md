# Package: startup

Application initialization and environment configuration from files.

## Usage

```go
import "github.com/amp-labs/amp-common/startup"

func main() {
    // Load env vars from ENV_FILE
    err := startup.ConfigureEnvironment()

    // Or specify files explicitly
    err := startup.ConfigureEnvironmentFromFiles(
        []string{".env", ".env.local"},
        startup.WithAllowOverride(true),
    )

    // Enable env var access auditing
    err := startup.ConfigureEnvironment(
        startup.WithEnableRecording(true),
        startup.WithAuditLogFile("env-audit.jsonl"),
    )
}
```

## Common Patterns

- `ConfigureEnvironment()` - Load from ENV_FILE (semicolon-separated paths)
- `ConfigureEnvironmentFromFiles()` - Load from specified files
- Options: WithAllowOverride, WithEnableRecording, WithAuditLogFile
- Auto-discovers ENV_DEBUG for debugging
- Recording/observation integration with envutil

## Gotchas

- ENV_FILE can contain multiple paths separated by semicolons
- Default: existing env vars take precedence (unless WithAllowOverride)
- Recording has minimal overhead but should be disabled in production
- Audit logs flushed every 5 seconds

## Related

- `envutil` - Environment variable parsing
- `logger` - Logging integration
