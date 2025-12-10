// Package should provides utilities for cleanup operations that should succeed
// but may fail in practice. Instead of returning errors, these functions log
// failures, making them suitable for defer statements and cleanup code.
package should

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/amp-labs/amp-common/logger"
)

// Close attempts to close the given io.Closer and logs an error if it fails.
// This is useful for cleanup in defer statements where you want to ensure
// resources are closed but don't want to complicate error handling.
//
// The args parameter is optional and can be used in three ways:
//   - No args: uses a default error message
//   - One arg: treated as the error message
//   - Multiple args: first arg is a format string, remaining args are formatting values
//
// Examples:
//
//	defer should.Close(file)                                    // uses default message
//	defer should.Close(file, "failed to close file")            // simple message
//	defer should.Close(file, "failed to close %s", filename)    // formatted message
func Close(closer io.Closer, args ...any) {
	err := closer.Close()
	if err != nil {
		msg := argsToMessage(args)
		if msg == "" {
			logger.Get().Error("error closing io.Closer", "error", err)
		} else {
			logger.Get().Error(msg, "error", err)
		}
	}
}

// argsToMessage converts variadic args into a formatted message string.
// Returns empty string if no args, Sprint if one arg, or Sprintf if multiple args.
func argsToMessage(args []any) string {
	if len(args) == 0 {
		return ""
	}

	if len(args) == 1 {
		return fmt.Sprint(args[0])
	}

	fmtStr := fmt.Sprint(args[0])
	remaining := args[1:]

	return fmt.Sprintf(fmtStr, remaining...)
}

// Remove attempts to remove the named file or directory and logs an error if it fails.
// This is useful for cleanup operations where you want to ensure temporary files
// are removed but don't want to complicate error handling.
//
// Example:
//
//	defer should.Remove("/tmp/tempfile", "failed to remove temp file")
func Remove(path string, msg string) {
	err := os.Remove(path)
	if err != nil {
		slog.Error(msg, "error", err)
	}
}
