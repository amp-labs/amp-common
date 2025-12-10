// Package cli provides terminal interaction utilities including banners, dividers, and user prompts.
//
// Banner and Divider functions create formatted output using Unicode box-drawing characters.
// Prompt functions provide interactive user input with validation.
// MultiSelect enables interactive multi-choice selection menus.
//
// Set the environment variable AMP_NO_BANNER=true to suppress banner boxes.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
	"github.com/amp-labs/amp-common/should"
)

const (
	boxTopLeft     = "╒"
	boxBottomLeft  = "└"
	boxTopRight    = "╕"
	boxBottomRight = "┘"
	boxSide        = "│"
	boxTop         = "═"
	boxBottom      = "─"
	dividerLeft    = "┠"
	dividerMiddle  = "─"
	dividerRight   = "┨"
	ellipsis       = "…"
)

const (
	AlignLeft = iota
	AlignCenter
	AlignRight
)

const (
	bannerPadding   = 2
	dividerPadding  = 2
	truncateReserve = 1
	halfDivisor     = 2
)

var suppressBanner = lazy.NewCtx[bool](func(ctx context.Context) bool {
	return envutil.Bool(ctx, "AMP_NO_BANNER",
		envutil.Default(false)).
		ValueOrElse(false)
})

const DefaultTerminalWidth = 80

// DividerAutoWidth creates a horizontal divider line that spans the terminal width.
// Auto-detects the terminal width or falls back to DefaultTerminalWidth if detection fails.
func DividerAutoWidth() string {
	_, w, e := TerminalDimensions()
	if e != nil || w == 0 {
		w = DefaultTerminalWidth
	}

	return Divider(int(w)) //nolint:gosec // Terminal width is bounded by screen size, no overflow risk
}

// BannerAutoWidth creates a formatted banner with auto-detected terminal width.
// The banner is drawn with Unicode box characters and can align text left, center, or right.
// Set AMP_NO_BANNER=true to suppress the banner box and return just the text with a newline.
// Parameters:
//   - s: The text to display (can include newlines for multi-line banners)
//   - a: Alignment constant (AlignLeft, AlignCenter, or AlignRight)
func BannerAutoWidth(ctx context.Context, s string, a int) string {
	if suppressBanner.Get(ctx) {
		return s + "\n"
	}

	_, w, e := TerminalDimensions()
	if e != nil || w == 0 {
		w = DefaultTerminalWidth
	}

	return Banner(ctx, s, int(w), a) //nolint:gosec // Terminal width is bounded by screen size, no overflow risk
}

// Divider creates a horizontal divider line with the specified width.
// Uses Unicode box-drawing characters (┠─┨).
func Divider(width int) string {
	return fmt.Sprintf("%s%s%s\n", dividerLeft, strings.Repeat(dividerMiddle, width-dividerPadding), dividerRight)
}

// Banner creates a formatted text banner with the specified width and alignment.
// The banner is drawn with Unicode box characters (╒═╕ for top, └─┘ for bottom, │ for sides).
// Text longer than the width is truncated with an ellipsis (…).
// Set AMP_NO_BANNER=true to suppress the banner box and return just the text with a newline.
// Parameters:
//   - s: The text to display (can include newlines for multi-line banners)
//   - width: The total width of the banner in characters
//   - alignment: Alignment constant (AlignLeft, AlignCenter, or AlignRight)
func Banner(ctx context.Context, s string, width int, alignment int) string {
	if suppressBanner.Get(ctx) {
		return s + "\n"
	}

	lines := getLines(s)
	if len(lines) == 0 {
		return ""
	}

	if width <= 0 {
		return ""
	}

	dividerTop := fmt.Sprintf("%s%s%s", boxTopLeft, strings.Repeat(boxTop, width-bannerPadding), boxTopRight)
	parts := []string{dividerTop}

	for _, l := range lines {
		var line string

		switch alignment {
		case AlignCenter:
			line = padCenter(l, width-bannerPadding)
		case AlignLeft:
			line = padLeft(l, width-bannerPadding)
		case AlignRight:
			line = padRight(l, width-bannerPadding)
		default:
			return ""
		}

		parts = append(parts, fmt.Sprintf("%s%s%s", boxSide, line, boxSide))
	}

	dividerBottom := fmt.Sprintf("%s%s%s", boxBottomLeft, strings.Repeat(boxBottom, width-bannerPadding), boxBottomRight)
	parts = append(parts, dividerBottom)

	return strings.Join(parts, "\n")
}

// getLines splits text into lines, normalizing line endings.
func getLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")

	return strings.Split(s, "\n")
}

// countGraphic counts the number of visible (graphic) characters in a string.
// This is used for accurate width calculation when padding, as it ignores control characters.
func countGraphic(s string) int {
	count := 0

	for _, r := range s {
		if unicode.IsGraphic(r) {
			count++
		}
	}

	return count
}

// truncateGraphic truncates a string to n graphic characters.
// Returns the truncated string and the actual count of graphic characters in it.
func truncateGraphic(s string, n int) (string, int) {
	out := ""
	count := 0

	var outSb170 strings.Builder

	for _, r := range s {
		if unicode.IsGraphic(r) {
			count++
		}

		if count >= n {
			break
		}

		outSb170.WriteRune(r)
	}

	out += outSb170.String()

	return out, count
}

// padCenter pads text to the specified width with center alignment.
// Text longer than width is truncated with an ellipsis.
func padCenter(text string, width int) string {
	length := countGraphic(text)
	if length == width {
		return text
	}

	str := text
	if length > width {
		str, length = truncateGraphic(str, width-truncateReserve)
		str += ellipsis
	}

	spaces := func(n int) string {
		return strings.Repeat(" ", n)
	}

	diff := width - length
	leftPad := diff / halfDivisor
	rightPad := diff - leftPad

	return fmt.Sprintf("%s%s%s", spaces(leftPad), str, spaces(rightPad))
}

// padLeft pads text to the specified width with left alignment (text on left, padding on right).
// Text longer than width is truncated with an ellipsis.
func padLeft(text string, width int) string {
	length := countGraphic(text)
	if length == width {
		return text
	}

	str := text
	if length > width {
		str, length = truncateGraphic(str, width-truncateReserve)
		str += ellipsis
	}

	spaces := func(n int) string {
		return strings.Repeat(" ", n)
	}

	diff := width - length

	return fmt.Sprintf("%s%s", str, spaces(diff))
}

// padRight pads text to the specified width with right alignment (padding on left, text on right).
// Text longer than width is truncated with an ellipsis.
func padRight(text string, width int) string {
	length := countGraphic(text)
	if length == width {
		return text
	}

	str := text
	if length > width {
		str, length = truncateGraphic(str, width-truncateReserve)
		str += ellipsis
	}

	spaces := func(n int) string {
		return strings.Repeat(" ", n)
	}

	diff := width - length

	return fmt.Sprintf("%s%s", spaces(diff), str)
}

// size executes the 'stty size' command to get terminal dimensions.
// Returns a string in the format "rows columns".
func size() (string, error) {
	f, e := os.Open("/dev/tty")
	if e != nil {
		return "", e
	}

	defer should.Close(f, "closing /dev/tty")

	// Outputs: "rows columns"
	cmd := exec.CommandContext(context.Background(), "stty", "size")
	cmd.Stdin = f
	out, err := cmd.Output()

	return string(out), err
}

// parse parses the output from 'stty size' command.
// Expects input in the format "rows columns" and returns (rows, columns, error).
func parse(input string) (uint, uint, error) {
	parts := strings.Split(input, " ")

	rows, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	cols, err := strconv.Atoi(strings.Replace(parts[1], "\n", "", 1))
	if err != nil {
		return 0, 0, err
	}

	return uint(rows), uint(cols), nil //nolint:gosec // Terminal dimensions are small positive integers, no overflow risk
}

// TerminalDimensions returns (rows, cols, err).
//
//nolint:contextcheck // Terminal size query is instantaneous and doesn't benefit from context
func TerminalDimensions() (uint, uint, error) {
	output, err := size()
	if err != nil {
		return 0, 0, err
	}

	return parse(output)
}
