package envtypes

import (
	"os"

	"github.com/amp-labs/amp-common/tuple"
)

// LocalPath represents a filesystem path with its associated file information.
// It is commonly used for parsing file paths from environment variables.
type LocalPath struct {
	Path string
	Info os.FileInfo
}

// AsTuple converts the LocalPath to a Tuple2 of path and file info.
func (lp LocalPath) AsTuple() tuple.Tuple2[string, os.FileInfo] {
	return tuple.NewTuple2[string, os.FileInfo](lp.Path, lp.Info)
}
