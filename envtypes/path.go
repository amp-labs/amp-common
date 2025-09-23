package envtypes

import (
	"os"

	"github.com/amp-labs/amp-common/tuple"
)

type LocalPath struct {
	Path string
	Info os.FileInfo
}

func (lp LocalPath) AsTuple() tuple.Tuple2[string, os.FileInfo] {
	return tuple.NewTuple2[string, os.FileInfo](lp.Path, lp.Info)
}
