// Package debug provides debugging utilities for local development only (not for production use).
package debug

import (
	"context"
	"encoding/json"
	"io"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/logger"
)

// DumpContext inspects and dumps the context hierarchy as formatted JSON to the given writer.
func DumpContext(ctx context.Context, w io.Writer) {
	result := contexts.InspectContext(ctx)

	DumpJSON(result, w)
}

// DumpJSON dumps the given value as JSON to the given writer.
func DumpJSON(v any, w io.Writer) {
	encoder := json.NewEncoder(w)

	// JSON may have URLs with special symbols which shouldn't be escaped. Ex: `&`.
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(v)
	if err != nil {
		logger.Fatal("error marshaling to JSON: %w", "error", err)
	}
}
