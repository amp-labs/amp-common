//go:build !unix

package diskmap

import (
	"fmt"
	"os"
	"path/filepath"
)

// lockDir opens the lock file. On this platform it does not stop a second
// map from opening the same directory.
func lockDir(dir string) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(dir, lockFile), os.O_RDWR|os.O_CREATE, filePerm) //nolint:gosec,lll // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: opening lock file: %w", err)
	}

	return file, nil
}
