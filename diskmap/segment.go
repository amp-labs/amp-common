package diskmap

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
)

const (
	segmentSuffix    = ".seg" // puts and tombstones
	compactionSuffix = ".cmp" // records moved by compaction
)

// segment is one append-only file of records. Foreground segments hold puts
// and tombstones in sequence order; compaction segments hold moves in move
// order. Recovery replays the two streams side by side.
//
// Segments are reference counted so reads can run without any lock: the log
// holds one reference while the segment is live, and readers take another
// while they hold a shard lock whose index points into the segment. The file
// is closed, and deleted if the segment was dropped, when the last reference
// goes.
type segment struct {
	id         uint32
	compaction bool
	file       *os.File
	path       string

	// written counts bytes already in the file. It only grows, so readers
	// can check it without the log lock.
	written atomic.Int64

	// size counts every byte appended, including bytes still in a write
	// buffer; dead counts bytes of records that were overwritten, removed or
	// relocated; synced counts bytes known to be on stable storage. All are
	// guarded by the log lock.
	size   int64
	dead   int64
	synced int64

	refs   atomic.Int32
	doomed atomic.Bool
}

func segmentPath(dir string, id uint32, compaction bool) string {
	suffix := segmentSuffix
	if compaction {
		suffix = compactionSuffix
	}

	return filepath.Join(dir, fmt.Sprintf("%010d%s", id, suffix))
}

func newSegment(id uint32, compaction bool, file *os.File, path string) *segment {
	s := &segment{id: id, compaction: compaction, file: file, path: path}
	s.refs.Store(1)

	return s
}

func createSegment(dir string, segID uint32, compaction bool) (*segment, error) {
	path := segmentPath(dir, segID, compaction)

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerm) //nolint:gosec // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: creating segment: %w", err)
	}

	return newSegment(segID, compaction, file, path), nil
}

// openSegment opens an existing segment file. Its size, written and synced
// counts start at the file's length.
func openSegment(dir string, segID uint32, compaction bool) (*segment, error) {
	path := segmentPath(dir, segID, compaction)

	file, err := os.OpenFile(path, os.O_RDWR, filePerm) //nolint:gosec // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: opening segment: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("diskmap: opening segment: %w", err)
	}

	s := newSegment(segID, compaction, file, path)
	s.size, s.synced = info.Size(), info.Size()
	s.written.Store(info.Size())

	return s, nil
}

// truncate cuts the segment's file to size bytes.
func (s *segment) truncate(size int64) error {
	err := s.file.Truncate(size)
	if err != nil {
		return fmt.Errorf("diskmap: truncating segment %d: %w", s.id, err)
	}

	s.size, s.synced = size, min(s.synced, size)
	s.written.Store(size)

	return nil
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
