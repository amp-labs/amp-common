package diskmap

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
)

const segmentSuffix = ".seg"

// segment is one append-only file of records.
//
// Segments are reference counted so reads can run without any lock: the log
// holds one reference while the segment is live, and readers take another
// while they hold a shard lock whose index points into the segment. The file
// is closed, and deleted if the segment was dropped, when the last reference
// goes.
type segment struct {
	id   uint32
	file *os.File
	path string

	// written counts bytes already in the file. It only grows, so readers
	// can check it without the log lock.
	written atomic.Int64

	// size counts every byte appended, including bytes still in a write
	// buffer; dead counts bytes of records that were overwritten, removed or
	// relocated. Both are guarded by the log lock.
	size     int64
	dead     int64
	unsynced bool

	refs   atomic.Int32
	doomed atomic.Bool
}

func segmentPath(dir string, id uint32) string {
	return filepath.Join(dir, fmt.Sprintf("%010d%s", id, segmentSuffix))
}

func newSegment(id uint32, file *os.File, path string) *segment {
	s := &segment{id: id, file: file, path: path}
	s.refs.Store(1)

	return s
}

func createSegment(dir string, segID uint32) (*segment, error) {
	path := segmentPath(dir, segID)

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerm) //nolint:gosec // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: creating segment: %w", err)
	}

	return newSegment(segID, file, path), nil
}

func openSegment(dir string, segID uint32, size, dead int64) (*segment, error) {
	path := segmentPath(dir, segID)

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

	s := newSegment(segID, file, path)
	s.size, s.dead = size, dead
	s.written.Store(size)

	return s, nil
}

func (s *segment) acquire() {
	s.refs.Add(1)
}

// release drops a reference, closing the file (and deleting it if the
// segment was dropped) when it was the last.
func (s *segment) release() {
	if s.refs.Add(-1) != 0 {
		return
	}

	_ = s.file.Close()

	if s.doomed.Load() {
		_ = os.Remove(s.path)
	}
}

// drop releases the log's reference to a segment that is no longer part of
// the map, deleting its file once no reader is using it.
func (s *segment) drop() {
	s.doomed.Store(true)
	s.release()
}

func (s *segment) sync() error {
	if !s.unsynced {
		return nil
	}

	err := s.file.Sync()
	if err != nil {
		return fmt.Errorf("diskmap: syncing segment: %w", err)
	}

	s.unsynced = false

	return nil
}

func readFile(s *segment, off int64, buf []byte) error {
	_, err := s.file.ReadAt(buf, off)
	if err != nil {
		return fmt.Errorf("diskmap: reading segment %d: %w", s.id, err)
	}

	return nil
}

// segTable maps segment ids to segments. It is replaced, never modified, so
// readers can use it without a lock.
type segTable struct {
	base uint32
	segs []*segment // segs[id-base]; nil once dropped
}

func (t *segTable) get(id uint32) *segment {
	if id < t.base || id-t.base >= uint32(len(t.segs)) { //nolint:gosec // Segment counts fit in uint32.
		return nil
	}

	return t.segs[id-t.base]
}

// all returns the live segments in id order, which is the order they were
// written.
func (t *segTable) all() []*segment {
	out := make([]*segment, 0, len(t.segs))

	for _, s := range t.segs {
		if s != nil {
			out = append(out, s)
		}
	}

	return out
}

// with returns a copy of the table with s added. Ids must increase.
func (t *segTable) with(s *segment) *segTable {
	return newSegTable(append(t.all(), s))
}

// without returns a copy of the table with segment id removed.
func (t *segTable) without(id uint32) *segTable {
	return newSegTable(slices.DeleteFunc(t.all(), func(s *segment) bool { return s.id == id }))
}

// newSegTable builds a table from live segments in ascending id order.
func newSegTable(live []*segment) *segTable {
	if len(live) == 0 {
		return &segTable{}
	}

	base := live[0].id
	segs := make([]*segment, live[len(live)-1].id-base+1)

	for _, s := range live {
		segs[s.id-base] = s
	}

	return &segTable{base: base, segs: segs}
}
