package utils //nolint:revive // utils is an appropriate package name for utility functions

import "regexp"

// GrepChannel creates a filtered channel pair using a POSIX regex pattern.
// Messages sent to the write channel are matched against the regex, and only
// matching messages are forwarded to the read channel.
// Returns (write channel, read channel, error).
// The read channel is automatically closed when the write channel is closed.
func GrepChannel(expr string) (in chan<- string, out <-chan string, err error) {
	re, err := regexp.CompilePOSIX(expr)
	if err != nil {
		return nil, nil, err
	}

	w := make(chan string)
	r := make(chan string)

	go func() {
		for msg := range w {
			if re.MatchString(msg) {
				r <- msg
			}
		}

		close(r)
	}()

	return w, r, nil
}
