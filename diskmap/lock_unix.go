//go:build unix

package diskmap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	lockWait  = 250 * time.Millisecond
	lockRetry = 5 * time.Millisecond
)

// lockDir takes an exclusive lock on dir, so two maps never use one
// directory at once. The operating system releases it if the process dies.
func lockDir(dir string) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(dir, lockFile), os.O_RDWR|os.O_CREATE, filePerm) //nolint:gosec,lll // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: opening lock file: %w", err)
	}

	// A process that forks briefly shares its descriptors with the child
	// until the child execs, which can keep a just-released lock held for
	// an instant, so retry for a short while before giving up.
	deadline := time.Now().Add(lockWait)

	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) //nolint:gosec // Descriptors fit in int.
		if !errors.Is(err, syscall.EWOULDBLOCK) || time.Now().After(deadline) {
			break
		}

		time.Sleep(lockRetry)
	}

	if err != nil {
		_ = file.Close()

		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%w: %s", ErrLocked, dir)
		}

		return nil, fmt.Errorf("diskmap: locking directory: %w", err)
	}

	return file, nil
}
