// Package should provides utilities for cleanup operations that should succeed
// but may fail in practice. Instead of returning errors, these functions log
// failures, making them suitable for defer statements and cleanup code.
package should

import (
	"io"
	"log/slog"
	"os"
)

// Close attempts to close the given io.Closer and logs an error if it fails.
// This is useful for cleanup in defer statements where you want to ensure
// resources are closed but don't want to complicate error handling.
//
// Example:
//
//	defer should.Close(file, "failed to close file")
func Close(closer io.Closer, msg string) {
	if err := closer.Close(); err != nil {
		slog.Error(msg, "error", err)
	}
}

// Remove attempts to remove the named file or directory and logs an error if it fails.
// This is useful for cleanup operations where you want to ensure temporary files
// are removed but don't want to complicate error handling.
//
// Example:
//
//	defer should.Remove("/tmp/tempfile", "failed to remove temp file")
func Remove(path string, msg string) {
	if err := os.Remove(path); err != nil {
		slog.Error(msg, "error", err)
	}
}
