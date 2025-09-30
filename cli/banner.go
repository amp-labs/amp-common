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
	ll             = "┠"
	mm             = "─"
	rr             = "┨"
)

const (
	AlignLeft = iota
	AlignCenter
	AlignRight
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
	return Divider(int(w))
}

func BannerAutoWidth(s string, a int) string {
	if suppressBanner.Get() {
		return s + "\n"
	}

	_, w, e := TerminalDimensions()
	if e != nil || w == 0 {
		w = DefaultTerminalWidth
	}
	return Banner(s, int(w), a)
}

func Divider(w int) string {
	return fmt.Sprintf("%s%s%s\n", ll, strings.Repeat(mm, w-2), rr)
}

func Banner(s string, w int, a int) string {
	if suppressBanner.Get() {
		return s + "\n"
	}

	lines := getLines(s)
	if len(lines) == 0 {
		return ""
	}
	if w <= 0 {
		return ""
	}

	dividerTop := fmt.Sprintf("%s%s%s", boxTopLeft, strings.Repeat(boxTop, w-2), boxTopRight)
	parts := []string{dividerTop}

	for _, l := range lines {
		if a == AlignCenter {
			line := padCenter(l, w-2)
			parts = append(parts, fmt.Sprintf("%s%s%s", boxSide, line, boxSide))
		} else if a == AlignLeft {
			line := padLeft(l, w-2)
			parts = append(parts, fmt.Sprintf("%s%s%s", boxSide, line, boxSide))
		} else if a == AlignRight {
			line := padRight(l, w-2)
			parts = append(parts, fmt.Sprintf("%s%s%s", boxSide, line, boxSide))
		} else {
			return ""
		}
	}

	dividerBottom := fmt.Sprintf("%s%s%s", boxBottomLeft, strings.Repeat(boxBottom, w-2), boxBottomRight)
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

func padCenter(s string, w int) string {
	length := countGraphic(s)

	if length == w {
		return s
	}
	if length > w {
		s, length = truncateGraphic(s, w-1)
		s += "…"
	}

	spaces := func(n int) string {
		return strings.Repeat(" ", n)
	}

	diff := w - length
	leftPad := diff / 2
	rightPad := diff - leftPad

	return fmt.Sprintf("%s%s%s", spaces(leftPad), s, spaces(rightPad))
}

func padLeft(s string, w int) string {
	length := countGraphic(s)

	if length == w {
		return s
	}
	if length > w {
		s, length = truncateGraphic(s, w-1)
		s += "…"
	}

	spaces := func(n int) string {
		return strings.Repeat(" ", n)
	}

	diff := w - length
	return fmt.Sprintf("%s%s", s, spaces(diff))
}

func padRight(s string, w int) string {
	length := countGraphic(s)

	if length == w {
		return s
	}
	if length > w {
		s, length = truncateGraphic(s, w-1)
		s += "…"
	}

	spaces := func(n int) string {
		return strings.Repeat(" ", n)
	}

	diff := w - length
	return fmt.Sprintf("%s%s", spaces(diff), s)
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
	return uint(rows), uint(cols), nil
}

// TerminalDimensions returns (rows, cols, err)
func TerminalDimensions() (uint, uint, error) {
	output, err := size()
	if err != nil {
		return 0, 0, err
	}
	return parse(output)
}
