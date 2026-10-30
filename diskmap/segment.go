package diskmap

import (
	"fmt"
	"os"
	"path/filepath"
)

const segmentSuffix = ".seg"

// segment is one append-only file of records.
type segment struct {
	id   uint32
	file *os.File
	path string

	// size counts every byte appended, including bytes still in the write
	// buffer. dead counts bytes of records that were overwritten or removed.
	size int64
	dead int64

	// unsynced is set once the segment is written and cleared by fsync.
	unsynced bool
}

func segmentPath(dir string, id uint32) string {
	return filepath.Join(dir, fmt.Sprintf("%010d%s", id, segmentSuffix))
}

func createSegment(dir string, id uint32) (*segment, error) {
	path := segmentPath(dir, id)

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerm) //nolint:gosec // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: creating segment: %w", err)
	}

	return &segment{id: id, file: file, path: path}, nil
}

func openSegment(dir string, id uint32, size, dead int64) (*segment, error) {
	path := segmentPath(dir, id)

	file, err := os.OpenFile(path, os.O_RDWR, filePerm) //nolint:gosec // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: opening segment: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("diskmap: opening segment: %w", err)
	}

	if info.Size() < size {
		_ = file.Close()

		return nil, fmt.Errorf("%w: segment %s is %d bytes, expected %d", ErrCorrupt, path, info.Size(), size)
	}

	return &segment{id: id, file: file, path: path, size: size, dead: dead}, nil
}

func (s *segment) sync() error {
	if !s.unsynced {
		return nil
	}

	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("diskmap: syncing segment: %w", err)
	}

	s.unsynced = false

	return nil
}

// remove closes and deletes the segment file.
func (s *segment) remove() error {
	closeErr := s.file.Close()

	if err := os.Remove(s.path); err != nil {
		return fmt.Errorf("diskmap: removing segment: %w", err)
	}

	if closeErr != nil {
		return fmt.Errorf("diskmap: closing segment: %w", closeErr)
	}

	return nil
}
