# Package: cli

Terminal interaction utilities with banners, dividers, and prompts.

## Usage

```go
import "github.com/amp-labs/amp-common/cli"

// Auto-width banners
banner := cli.BannerAutoWidth(ctx, "My Application", cli.AlignCenter)

// Dividers
divider := cli.DividerAutoWidth()

// User prompts
answer := cli.PromptYN(os.Stdin, "Continue?")
value := cli.Prompt(os.Stdin, "Enter value: ")

// Multi-select menu
selected := cli.MultiSelect(os.Stdin, "Choose options:", []string{"A", "B", "C"})
```

## Common Patterns

- `BannerAutoWidth()` / `Banner()` - Formatted banners with Unicode box drawing
- `DividerAutoWidth()` / `Divider()` - Horizontal dividers
- `Prompt()` - Basic user input
- `PromptYN()` - Yes/no questions
- `MultiSelect()` - Interactive multi-choice menus
- Set `AMP_NO_BANNER=true` to suppress banner boxes

## Gotchas

- Auto-detects terminal width (fallback: 80 cols)
- Uses Unicode box-drawing characters
- Banner text truncated with ellipsis if too long
