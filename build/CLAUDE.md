# Package: build

Parse build information embedded at compile time via -ldflags.

## Usage

```go
// In main.go of application
var buildInfoJSON string  // Set via -ldflags at compile time

func main() {
    if info, ok := build.Parse(buildInfoJSON); ok {
        fmt.Printf("Version: %s\n", info.GitCommit)
        fmt.Printf("Built: %s\n", info.BuildTime)
    }
}
```

## Common Patterns

- `Info` struct contains git info, build metadata, dependencies
- Populated via -ldflags: `-X 'main.buildInfoJSON={"git_commit":"abc123",...}'`
- Returns (nil, false) for empty or invalid JSON

## Gotchas

- This package only parses - injection happens in build scripts
- Used by applications consuming amp-common, not by amp-common itself
