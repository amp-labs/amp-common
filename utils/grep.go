package utils

import "regexp"

// Returns (caller write channel, caller read channel, err).
func GrepChannel(expr string) (chan string, chan string, error) {
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
