// Package diskmap provides a disk-backed map from string keys to byte-slice
// values, for maps that have outgrown memory.
//
// Values are appended to segment files and found through an on-disk hash
// index read through a fixed-size page cache, so memory use stays near the
// configured limit no matter how many entries the map holds. Every lookup
// reads exactly one index page (often from the cache) and one record. Growth
// splits one index page at a time and space is reclaimed by compacting a
// little at a time alongside writes, so no single operation pays for a full
// rehash or a full compaction.
//
// A map lives in its own directory. Close writes the state needed to reopen
// it; a directory whose map was not closed (for example after a crash) cannot
// be reopened, and Open returns ErrNotCleanlyClosed. The map is a place to
// keep data that does not fit in memory, not a durable database.
package diskmap

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/zeebo/xxh3"
)

var (
	// ErrClosed is returned by operations on a closed map.
	ErrClosed = errors.New("diskmap: map is closed")

	// ErrNotCleanlyClosed is returned by Open when the directory holds a map
	// that was never closed. Its contents cannot be trusted; reopen with
	// WithDiscardExisting to start over.
	ErrNotCleanlyClosed = errors.New("diskmap: map was not closed cleanly")

	// ErrCorrupt is returned when on-disk data fails validation.
	ErrCorrupt = errors.New("diskmap: corrupt data")

	// ErrKeyTooLarge is returned for keys longer than MaxKeySize.
	ErrKeyTooLarge = errors.New("diskmap: key too large")

	// ErrValueTooLarge is returned for values longer than MaxValueSize.
	ErrValueTooLarge = errors.New("diskmap: value too large")

	// ErrTooManyCollisions is returned when the index cannot split a page
	// because its keys' hashes do not differ. It does not happen with the
	// default hash.
	ErrTooManyCollisions = errors.New("diskmap: too many hash collisions")

	// ErrInvalidOption is returned by Open for out-of-range options.
	ErrInvalidOption = errors.New("diskmap: invalid option")
)

const (
	maxWriteBuffer = 1 << 20
	minWriteBuffer = 64 << 10
	chunkSize      = 64 << 10
)

// Map is a disk-backed map from string keys to byte-slice values. It is safe
// for concurrent use.
type Map struct {
	mu sync.Mutex

	dir  string
	opts options
	seed uint64
	hash func(key string, seed uint64) uint64

	idxFile *os.File
	pc      *pageCache
	ix      *index

	segs    map[uint32]*segment
	active  *segment
	nextSeg uint32
	wbuf    []byte
	flushed int64 // bytes of the active segment already written to its file

	count int
	total int64 // bytes in all segments
	dead  int64 // bytes of overwritten or removed records

	comp      compactor
	iterators int
	gen       uint64 // incremented by Clear, so iterators can tell
	closed    bool
	failed    error // a write failed and the files may be inconsistent

	keyBuf []byte
}

// Open opens the map stored in dir, creating the directory and an empty map
// if needed.
func Open(dir string, opts ...Option) (*Map, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	if err := o.validate(); err != nil {
		return nil, err
	}

	if o.hash == nil {
		o.hash = xxh3.HashStringSeed
	}

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("diskmap: creating directory: %w", err)
	}

	if o.discardExisting {
		if err := removeDataFiles(dir); err != nil {
			return nil, err
		}
	}

	saved, err := readMeta(dir)
	if err != nil {
		return nil, err
	}

	if saved == nil {
		files, err := dataFiles(dir)
		if err != nil {
			return nil, err
		}

		if len(files) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrNotCleanlyClosed, dir)
		}
	}

	wbufSize := min(max(o.memoryLimit/16, minWriteBuffer), maxWriteBuffer)

	m := &Map{
		dir:  dir,
		opts: o,
		hash: o.hash,
		segs: make(map[uint32]*segment),
		wbuf: make([]byte, 0, wbufSize),
	}

	idxFile, err := os.OpenFile(filepath.Join(dir, indexFile), os.O_RDWR|os.O_CREATE, filePerm) //nolint:gosec,lll // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: opening index: %w", err)
	}

	m.idxFile = idxFile
	m.pc = newPageCache(idxFile, (o.memoryLimit-wbufSize-chunkSize)/pageSize)

	if saved == nil {
		err = m.create()
	} else {
		err = m.load(saved)
	}

	if err != nil {
		m.closeFiles()

		return nil, err
	}

	return m, nil
}

func (m *Map) create() error {
	var seed [8]byte

	_, _ = rand.Read(seed[:])
	m.seed = binary.LittleEndian.Uint64(seed[:])

	ix, err := newIndex(m.pc)
	if err != nil {
		return err
	}

	m.ix = ix

	return m.rotate()
}

func (m *Map) load(saved *meta) error {
	m.seed = saved.seed
	m.count = int(saved.count) //nolint:gosec // Written from an int.
	m.nextSeg = saved.nextSeg
	m.ix = &index{pc: m.pc, dir: saved.dir, depth: saved.depth, numPages: saved.numPages}
	m.pc.setPages(saved.numPages)

	info, err := m.idxFile.Stat()
	if err != nil {
		return fmt.Errorf("diskmap: opening index: %w", err)
	}

	if info.Size() < int64(saved.numPages)*pageSize {
		return fmt.Errorf("%w: index file is shorter than its %d pages", ErrCorrupt, saved.numPages)
	}

	for _, sm := range saved.segments {
		s, err := openSegment(m.dir, sm.id, sm.size, sm.dead)
		if err != nil {
			return err
		}

		m.segs[s.id] = s
		m.total += s.size
		m.dead += s.dead
		m.active = s
	}

	m.flushed = m.active.size

	if err := m.active.file.Truncate(m.active.size); err != nil {
		return fmt.Errorf("diskmap: truncating active segment: %w", err)
	}

	return removeMeta(m.dir)
}

// Get returns a copy of the value stored under key.
func (m *Map) Get(key string) (value []byte, found bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.get(nil, key)
}

// AppendValue appends the value stored under key to dst and returns the
// extended slice. It lets callers reuse a buffer across reads; if the key is
// missing, dst is returned unchanged.
func (m *Map) AppendValue(dst []byte, key string) (value []byte, found bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dst == nil {
		dst = []byte{}
	}

	return m.get(dst, key)
}

// get appends the value for key to dst. A nil dst gets an exactly sized
// fresh slice.
func (m *Map) get(dst []byte, key string) ([]byte, bool, error) {
	if err := m.usable(); err != nil {
		return dst, false, err
	}

	h := m.hash(key, m.seed)

	frame, err := m.ix.frameFor(h)
	if err != nil {
		return dst, false, err
	}

	p := m.pc.buf(frame)

	for i := range pageCount(p) {
		if slotHash(p, i) != h {
			continue
		}

		out, ok, err := m.readValue(dst, slotLoc(p, i), key)
		if err != nil || ok {
			return out, ok, err
		}
	}

	return dst, false, nil
}

// readValue reads the record at l and, if it holds key, appends its value to
// dst. Records are read straight into dst's spare capacity and the value is
// then moved down over the header and key, so the value is copied once.
func (m *Map) readValue(dst []byte, l loc, key string) ([]byte, bool, error) {
	size := int(l.size)
	if size < recordSize(len(key), 0) {
		return dst, false, nil
	}

	if dst == nil {
		dst = make([]byte, 0, size)
	}

	start := len(dst)
	grown := slices.Grow(dst, size)
	buf := grown[start : start+size]

	if err := m.readAt(m.segs[l.seg], int64(l.off), buf); err != nil {
		return dst, false, err
	}

	k, v, err := parseRecord(buf, m.opts.verifyChecksums)
	if err != nil {
		return dst, false, fmt.Errorf("record at segment %d offset %d: %w", l.seg, l.off, err)
	}

	if string(k) != key {
		return dst, false, nil
	}

	n := copy(buf, v)

	return grown[:start+n], true, nil
}

// readAt fills buf with the bytes of segment s starting at off. The tail of
// the active segment may still be in the write buffer.
func (m *Map) readAt(s *segment, off int64, buf []byte) error {
	if s == nil {
		return fmt.Errorf("%w: index points to a missing segment", ErrCorrupt)
	}

	if s == m.active && off+int64(len(buf)) > m.flushed {
		split := max(m.flushed-off, 0)
		copy(buf[split:], m.wbuf[off+split-m.flushed:])
		buf = buf[:split]
	}

	if len(buf) == 0 {
		return nil
	}

	if _, err := s.file.ReadAt(buf, off); err != nil {
		return fmt.Errorf("diskmap: reading segment %d: %w", s.id, err)
	}

	return nil
}

// Contains reports whether key is in the map.
func (m *Map) Contains(key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.usable(); err != nil {
		return false, err
	}

	_, _, _, found, err := m.find(m.hash(key, m.seed), key)

	return found, err
}

// Add stores value under key, replacing any existing value. The map keeps its
// own copy of value.
func (m *Map) Add(key string, value []byte) error {
	if len(key) > MaxKeySize {
		return ErrKeyTooLarge
	}

	if len(value) > MaxValueSize {
		return ErrValueTooLarge
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.usable(); err != nil {
		return err
	}

	h := m.hash(key, m.seed)

	frame, slot, old, found, err := m.find(h, key)
	if err != nil {
		return err
	}

	l, err := m.appendRecord(key, value)
	if err != nil {
		return err
	}

	// Appending touches only segment files, so frame still holds the page.
	if found {
		setSlotLoc(m.pc.buf(frame), slot, l)
		m.pc.markDirty(frame)
		m.markDead(old)
	} else {
		if err := m.ix.insert(h, l); err != nil {
			m.markDead(l)

			return err
		}

		m.count++
	}

	return m.compactStep(int64(l.size))
}

// Remove deletes key from the map. Removing a missing key is not an error.
func (m *Map) Remove(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.usable(); err != nil {
		return err
	}

	frame, slot, old, found, err := m.find(m.hash(key, m.seed), key)
	if err != nil || !found {
		return err
	}

	removeSlot(m.pc.buf(frame), slot)
	m.pc.markDirty(frame)
	m.markDead(old)
	m.count--

	return m.compactStep(int64(old.size))
}

// find locates the index slot for key. The returned frame stays valid until
// the next index call.
func (m *Map) find(h uint64, key string) (frame, slot int, l loc, found bool, err error) {
	frame, err = m.ix.frameFor(h)
	if err != nil {
		return 0, 0, loc{}, false, err
	}

	p := m.pc.buf(frame)

	for i := range pageCount(p) {
		if slotHash(p, i) != h {
			continue
		}

		l = slotLoc(p, i)

		same, err := m.recordHasKey(l, key)
		if err != nil {
			return 0, 0, loc{}, false, err
		}

		if same {
			return frame, i, l, true, nil
		}
	}

	return frame, 0, loc{}, false, nil
}

// recordHasKey reports whether the record at l stores key, reading only its
// header and key.
func (m *Map) recordHasKey(l loc, key string) (bool, error) {
	n := recordSize(len(key), 0)
	if int(l.size) < n {
		return false, nil
	}

	if cap(m.keyBuf) < n {
		m.keyBuf = make([]byte, n)
	}

	buf := m.keyBuf[:n]
	if err := m.readAt(m.segs[l.seg], int64(l.off), buf); err != nil {
		return false, err
	}

	keyLen, _ := recordLengths(buf)

	return keyLen == len(key) && string(buf[recordHeaderSize:]) == key, nil
}

func (m *Map) markDead(l loc) {
	if s := m.segs[l.seg]; s != nil {
		s.dead += int64(l.size)
		m.dead += int64(l.size)
	}
}

// appendRecord writes a record for key and value to the active segment.
func (m *Map) appendRecord(key string, value []byte) (loc, error) {
	size := int64(recordSize(len(key), len(value)))

	if err := m.reserve(size); err != nil {
		return loc{}, err
	}

	if size <= int64(cap(m.wbuf)) {
		m.wbuf = appendRecord(m.wbuf, key, value)

		return m.appended(size), nil
	}

	// Too big to buffer; the buffer is empty after reserve, so write it now.
	rec := appendRecord(make([]byte, 0, size), key, value)
	if err := m.writeDirect(rec); err != nil {
		return loc{}, err
	}

	return m.appended(size), nil
}

// appendRaw copies an encoded record to the active segment.
func (m *Map) appendRaw(rec []byte) (loc, error) {
	size := int64(len(rec))

	if err := m.reserve(size); err != nil {
		return loc{}, err
	}

	if size <= int64(cap(m.wbuf)) {
		m.wbuf = append(m.wbuf, rec...)

		return m.appended(size), nil
	}

	if err := m.writeDirect(rec); err != nil {
		return loc{}, err
	}

	return m.appended(size), nil
}

// reserve makes room for a record of the given size: it starts a new segment
// if this one would grow past the segment size, and flushes the write buffer
// if the record does not fit.
func (m *Map) reserve(size int64) error {
	if m.active.size > 0 && m.active.size+size > m.opts.segmentSize {
		if err := m.rotate(); err != nil {
			return err
		}
	}

	if len(m.wbuf)+int(size) > cap(m.wbuf) {
		return m.flush()
	}

	return nil
}

func (m *Map) writeDirect(rec []byte) error {
	if _, err := m.active.file.WriteAt(rec, m.flushed); err != nil {
		return m.fail(fmt.Errorf("diskmap: writing segment %d: %w", m.active.id, err))
	}

	m.flushed += int64(len(rec))

	return nil
}

// appended records that size bytes were just added to the active segment and
// returns their location.
func (m *Map) appended(size int64) loc {
	off := m.active.size

	m.active.size += size
	m.active.unsynced = true
	m.total += size

	return loc{seg: m.active.id, off: uint32(off), size: uint32(size)} //nolint:gosec // Segments stay below 4 GiB.
}

// flush writes the write buffer to the active segment.
func (m *Map) flush() error {
	if len(m.wbuf) == 0 {
		return nil
	}

	if _, err := m.active.file.WriteAt(m.wbuf, m.flushed); err != nil {
		return m.fail(fmt.Errorf("diskmap: writing segment %d: %w", m.active.id, err))
	}

	m.flushed += int64(len(m.wbuf))
	m.wbuf = m.wbuf[:0]

	return nil
}

// rotate seals the active segment and starts a new one.
func (m *Map) rotate() error {
	if m.active != nil {
		if err := m.flush(); err != nil {
			return err
		}
	}

	s, err := createSegment(m.dir, m.nextSeg)
	if err != nil {
		return m.fail(err)
	}

	m.nextSeg++
	m.segs[s.id] = s
	m.active = s
	m.flushed = 0

	return nil
}

// fail marks the map unusable after a failed write left its files
// inconsistent.
func (m *Map) fail(err error) error {
	if m.failed == nil {
		m.failed = err
	}

	return err
}

func (m *Map) usable() error {
	if m.closed {
		return ErrClosed
	}

	if m.failed != nil {
		return m.failed
	}

	return m.pc.failed
}

// Size returns the number of entries in the map.
func (m *Map) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.count
}

// Clear removes every entry and deletes all data files.
func (m *Map) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.usable(); err != nil {
		return err
	}

	var errs []error

	for _, s := range m.segs {
		errs = append(errs, s.remove())
	}

	m.segs = make(map[uint32]*segment)
	m.active = nil
	m.wbuf = m.wbuf[:0]
	m.count, m.total, m.dead = 0, 0, 0
	m.comp.reset()
	m.gen++

	if err := m.idxFile.Truncate(0); err != nil {
		errs = append(errs, fmt.Errorf("diskmap: truncating index: %w", err))
	}

	if err := errors.Join(errs...); err != nil {
		return m.fail(err)
	}

	if err := m.ix.reset(); err != nil {
		return m.fail(err)
	}

	return m.rotate()
}

// Close writes everything needed to reopen the map, syncs it to disk and
// releases the map's files. Closing a closed map is a no-op.
func (m *Map) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	err := m.persist()

	return errors.Join(err, m.closeFiles())
}

func (m *Map) persist() error {
	if m.failed != nil {
		return m.failed
	}

	if err := m.flush(); err != nil {
		return err
	}

	if err := m.pc.flush(); err != nil {
		return err
	}

	if err := m.idxFile.Sync(); err != nil {
		return fmt.Errorf("diskmap: syncing index: %w", err)
	}

	saved := &meta{
		seed:     m.seed,
		count:    uint64(m.count), //nolint:gosec // Count is non-negative.
		numPages: m.ix.numPages,
		depth:    m.ix.depth,
		nextSeg:  m.nextSeg,
		dir:      m.ix.dir,
	}

	for _, id := range m.segmentIDs() {
		s := m.segs[id]
		if err := s.sync(); err != nil {
			return err
		}

		saved.segments = append(saved.segments, segmentMeta{id: s.id, size: s.size, dead: s.dead})
	}

	return writeMeta(m.dir, saved)
}

func (m *Map) closeFiles() error {
	var errs []error

	for _, s := range m.segs {
		errs = append(errs, s.file.Close())
	}

	if m.idxFile != nil {
		errs = append(errs, m.idxFile.Close())
	}

	return errors.Join(errs...)
}

// segmentIDs returns the ids of all segments in ascending order, which is
// the order they were written.
func (m *Map) segmentIDs() []uint32 {
	ids := make([]uint32, 0, len(m.segs))
	for id := range m.segs {
		ids = append(ids, id)
	}

	slices.Sort(ids)

	return ids
}

// Stats describes a map's size and cache behavior.
type Stats struct {
	// Entries is the number of keys in the map.
	Entries int
	// Segments is the number of segment files.
	Segments int
	// SegmentBytes is the total size of all segment files.
	SegmentBytes int64
	// DeadBytes is the part of SegmentBytes held by overwritten or removed
	// records, which compaction reclaims.
	DeadBytes int64
	// IndexPages is the number of pages in the index file.
	IndexPages int
	// CachedPages is the number of index pages the cache can hold.
	CachedPages int
	// PageHits, PageMisses and PageWrites count index cache lookups served
	// from memory, lookups that read the index file, and dirty pages written
	// back to it.
	PageHits   uint64
	PageMisses uint64
	PageWrites uint64
	// CompactedBytes is the number of segment bytes compaction has processed.
	CompactedBytes int64
}

// Stats returns a snapshot of the map's statistics.
func (m *Map) Stats() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()

	return Stats{
		Entries:        m.count,
		Segments:       len(m.segs),
		SegmentBytes:   m.total,
		DeadBytes:      m.dead,
		IndexPages:     int(m.ix.numPages),
		CachedPages:    m.pc.frames(),
		PageHits:       m.pc.hits,
		PageMisses:     m.pc.misses,
		PageWrites:     m.pc.writes,
		CompactedBytes: m.comp.processed,
	}
}
