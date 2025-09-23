package should

import (
	"io"
	"log/slog"
)

func Close(closer io.Closer, msg string) {
	if err := closer.Close(); err != nil {
		slog.Error(msg, "error", err)
	}
}
