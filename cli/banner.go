package cli

import (
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

	bannerPadding   = 2
	dividerPadding  = 2
	truncateReserve = 1
	halfDivisor     = 2
)

var suppressBanner = lazy.New[bool](func() bool {
	return envutil.Bool("AMP_NO_BANNER",
		envutil.Default(false)).
		ValueOrElse(false)
})

const DefaultTerminalWidth = 80

func DividerAutoWidth() string {
	_, w, e := TerminalDimensions()
	if e != nil || w == 0 {
		w = DefaultTerminalWidth
	}

	return Divider(int(w)) //nolint:gosec // Terminal width is bounded by screen size, no overflow risk
}

func BannerAutoWidth(s string, a int) string {
	if suppressBanner.Get() {
		return s + "\n"
	}

	_, w, e := TerminalDimensions()
	if e != nil || w == 0 {
		w = DefaultTerminalWidth
	}

	return Banner(s, int(w), a) //nolint:gosec // Terminal width is bounded by screen size, no overflow risk
}

func Divider(width int) string {
	return fmt.Sprintf("%s%s%s\n", dividerLeft, strings.Repeat(dividerMiddle, width-dividerPadding), dividerRight)
}

func Banner(s string, width int, alignment int) string {
	if suppressBanner.Get() {
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

func getLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")

	return strings.Split(s, "\n")
}

func countGraphic(s string) int {
	count := 0

	for _, r := range s {
		if unicode.IsGraphic(r) {
			count++
		}
	}

	return count
}

func truncateGraphic(s string, n int) (string, int) {
	out := ""
	count := 0

	for _, r := range s {
		if unicode.IsGraphic(r) {
			count++
		}

		if count >= n {
			break
		}

		out += string(r)
	}

	return out, count
}

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

func size() (string, error) {
	f, e := os.Open("/dev/tty")
	if e != nil {
		return "", e
	}

	defer should.Close(f, "closing /dev/tty")

	// Outputs: "rows columns"
	cmd := exec.Command("stty", "size")
	cmd.Stdin = f
	out, err := cmd.Output()

	return string(out), err
}

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
func TerminalDimensions() (uint, uint, error) {
	output, err := size()
	if err != nil {
		return 0, 0, err
	}

	return parse(output)
}
