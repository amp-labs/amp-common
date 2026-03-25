# Package: emoji

Emoji constants for terminal output and UI.

## Usage

```go
import "github.com/amp-labs/amp-common/emoji"

fmt.Printf("%s Starting server...\n", emoji.Rocket)
fmt.Printf("%s Warning: deprecated\n", emoji.Warning)
fmt.Printf("%s Task complete\n", emoji.Checkmark)
```

## Common Patterns

- Terminal output for CLI tools
- Status indicators (Ok, NotOk, Checkmark)
- Visual markers (Fire, ThumbsUp, Warning)

## Available

Rocket, Fire, ThumbsUp, Warning, Checkmark, Ok, NotOk, Robot, Construction, Stop, and many more (see emoji.go)

## Gotchas

- Terminal must support Unicode emoji rendering
