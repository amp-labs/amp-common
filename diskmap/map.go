// Package diskmap provides a disk-backed map from string keys to byte-slice
// values, for maps that have outgrown memory.
//
// Values are appended to segment files, and an on-disk hash index, read
// through a fixed-size page cache, says where each one is. Memory use stays
// near the configured limit no matter how many entries the map holds.
//
// Every lookup reads exactly one index page (usually from the cache) and one
// record. The index is split into independently locked shards, and record
// reads happen outside any lock, so goroutines rarely wait on each other.
// Growth splits one index page at a time, and background goroutines flush
// writes and compact dead space in small steps, so no single operation pays
// for a rehash, a flush or a compaction pass.
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
	"slices"
	"sync"
	"sync/atomic"

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
//
// Locks are taken in a fixed order: a shard lock, then the log lock. Reads
// hold only a shard lock, and only while consulting the index.
type Map struct {
	dir  string
	opts options
	seed uint64
	hash func(key string, seed uint64) uint64

	shards []*shard
	mask   uint64
	log    *valueLog

	closed  atomic.Bool
	closeMu sync.Mutex
	wg      sync.WaitGroup // flusher and compactor
	writers sync.WaitGroup // index page writers, one per shard
}

// Open opens the map stored in dir, creating the directory and an empty map
// if needed. The map must be closed with Close.
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
	} else {
		o.shards = len(saved.shards)
	}

	// Two write buffers and the compactor's chunk come out of the budget;
	// the rest holds index pages, split evenly between shards.
	bufCap := min(max(o.memoryLimit/32, minWriteBuffer), maxWriteBuffer)
	pages := (o.memoryLimit - 2*bufCap - chunkSize) / pageSize / o.shards

	m := &Map{
		dir:  dir,
		opts: o,
		hash: o.hash,
		mask: uint64(o.shards - 1), //nolint:gosec // Validated positive.
		log:  newValueLog(dir, o.segmentSize, bufCap, o.compactionThreshold),
	}

	if err := m.open(saved, pages); err != nil {
		m.releaseFiles()

		return nil, err
	}

	m.wg.Add(2) //nolint:mnd // Flusher and compactor.

	go m.log.flushLoop(&m.wg)
	go m.compactLoop()

	for _, sh := range m.shards {
		m.writers.Add(1)

		go sh.pc.writeLoop(&m.writers)
	}

	return m, nil
}

func (m *Map) open(saved *meta, pages int) error {
	for i := range m.opts.shards {
		sh, err := openShard(m.dir, i, pages)
		if err != nil {
			return err
		}

		m.shards = append(m.shards, sh)
	}

	if saved == nil {
		var seed [8]byte

		_, _ = rand.Read(seed[:])
		m.seed = binary.LittleEndian.Uint64(seed[:])

		for _, sh := range m.shards {
			if err := sh.create(); err != nil {
				return err
			}
		}

		return m.log.rotate()
	}

	m.seed = saved.seed

	for i, sh := range m.shards {
		if err := sh.load(saved.shards[i]); err != nil {
			return err
		}
	}

	if err := m.log.load(saved); err != nil {
		return err
	}

	return removeMeta(m.dir)
}

func (m *Map) shardFor(h uint64) *shard {
	return m.shards[h&m.mask]
}

// Get returns a copy of the value stored under key.
func (m *Map) Get(key string) (value []byte, found bool, err error) {
	return m.get(nil, key)
}

// AppendValue appends the value stored under key to dst and returns the
// extended slice. It lets callers reuse a buffer across reads; if the key is
// missing, dst is returned unchanged.
func (m *Map) AppendValue(dst []byte, key string) (value []byte, found bool, err error) {
	if dst == nil {
		dst = []byte{}
	}

	return m.get(dst, key)
}

// get appends the value for key to dst; a nil dst gets an exactly sized
// fresh slice. The index lookup happens under the shard lock and the record
// read outside it.
func (m *Map) get(dst []byte, key string) ([]byte, bool, error) {
	h := m.hash(key, m.seed)
	sh := m.shardFor(h)

	sh.mu.Lock()

	var (
		l   loc
		n   int
		err error
	)

	for {
		if err = sh.usable(m.log); err == nil {
			l, n, err = sh.candidates(h)
		}

		if err != errRetry { //nolint:errorlint // errRetry is never wrapped.
			break
		}
	}

	if err != nil || n == 0 {
		sh.mu.Unlock()

		return dst, false, err
	}

	if n > 1 {
		// Distinct keys with equal 64-bit hashes: rare enough to handle the
		// simple way, entirely under the lock (the page is cached now).
		defer sh.mu.Unlock()

		return m.getCollided(sh, dst, key, h)
	}

	// The index points into s, so s cannot be dropped while the shard is
	// locked; the reference keeps it open after the lock is released.
	s := m.log.segment(l.seg)
	if s == nil {
		sh.mu.Unlock()

		return dst, false, fmt.Errorf("%w: index points to missing segment %d", ErrCorrupt, l.seg)
	}

	s.acquire()
	sh.mu.Unlock()

	defer s.release()

	if dst == nil {
		dst = make([]byte, 0, l.size)
	}

	start := len(dst)
	grown := slices.Grow(dst, int(l.size))
	buf := grown[start : start+int(l.size)]

	if err := m.log.read(s, int64(l.off), buf); err != nil {
		return dst, false, err
	}

	n, ok, err := m.extractValue(buf, key, l)
	if err != nil || !ok {
		return dst, false, err
	}

	return grown[:start+n], true, nil
}

// getCollided is get for a key whose hash matches several slots. It runs
// under the shard lock.
func (m *Map) getCollided(sh *shard, dst []byte, key string, h uint64) ([]byte, bool, error) {
	frame, err := sh.ix.frameFor(h)
	if err != nil {
		return dst, false, err
	}

	p := sh.pc.buf(frame)
	mt := matchSlots(p, h)

	for i := mt.next(); i >= 0; i = mt.next() {
		l := slotLoc(p, i)

		s := m.log.segment(l.seg)
		if s == nil {
			return dst, false, fmt.Errorf("%w: index points to missing segment %d", ErrCorrupt, l.seg)
		}

		buf := make([]byte, l.size)
		if err := m.log.read(s, int64(l.off), buf); err != nil {
			return dst, false, err
		}

		n, ok, err := m.extractValue(buf, key, l)
		if err != nil {
			return dst, false, err
		}

		if ok {
			if dst == nil {
				dst = make([]byte, 0, n)
			}

			return append(dst, buf[:n]...), true, nil
		}
	}

	return dst, false, nil
}

// extractValue checks that rec holds key and moves its value to the front of
// rec, returning the value's length.
func (m *Map) extractValue(rec []byte, key string, l loc) (int, bool, error) {
	k, v, err := parseRecord(rec, m.opts.verifyChecksums)
	if err != nil {
		return 0, false, fmt.Errorf("record at segment %d offset %d: %w", l.seg, l.off, err)
	}

	if string(k) != key {
		return 0, false, nil
	}

	return copy(rec, v), true, nil
}

// Contains reports whether key is in the map.
func (m *Map) Contains(key string) (bool, error) {
	h := m.hash(key, m.seed)
	sh := m.shardFor(h)

	var checkBuf [2]keyCheck

	checks := checkBuf[:0]

	sh.mu.Lock()
	defer sh.mu.Unlock()

	for {
		if err := sh.usable(m.log); err != nil {
			return false, err
		}

		_, _, _, found, err := sh.find(m.log, h, key, &checks)
		if err != errRetry { //nolint:errorlint // errRetry is never wrapped.
			return found, err
		}
	}
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

	if m.closed.Load() {
		return ErrClosed
	}

	h := m.hash(key, m.seed)
	sh := m.shardFor(h)

	// Append first, then point the index at the new record. Concurrent Adds
	// of one key may append in one order and index in the other; either
	// result is a valid outcome of the race.
	m.log.writeMu.RLock()

	l, gen, err := m.log.appendRecord(key, value)
	if err != nil {
		m.log.writeMu.RUnlock()

		return err
	}

	var checkBuf [2]keyCheck

	checks := checkBuf[:0]

	sh.mu.Lock()

	dead, err := m.indexRecord(sh, h, key, l, gen, &checks)
	for err == errRetry { //nolint:errorlint // errRetry is never wrapped.
		dead, err = m.indexRecord(sh, h, key, l, gen, &checks)
	}

	sh.mu.Unlock()

	m.log.writeMu.RUnlock()

	if dead.size > 0 {
		m.log.markDead(dead.seg, dead.size)
	}

	return err
}

// indexRecord points key at the record l, appended in generation gen, and
// returns the location of the record that no longer holds a live value: the
// one l replaced, or l itself if it could not be indexed. It returns errRetry
// if it had to release the lock.
func (m *Map) indexRecord(sh *shard, h uint64, key string, l loc, gen uint64, checks *[]keyCheck) (loc, error) {
	if err := sh.usable(m.log); err != nil {
		return l, err
	}

	// Clear ran after the append and deleted the record with everything else.
	if m.log.gen.Load() != gen {
		return loc{}, nil
	}

	frame, slot, old, found, err := sh.find(m.log, h, key, checks)
	if err == errRetry { //nolint:errorlint // errRetry is never wrapped.
		return loc{}, err
	}

	if err != nil {
		return l, err
	}

	if found {
		setSlotLoc(sh.pc.buf(frame), slot, l)
		sh.pc.markDirty(frame)

		return old, nil
	}

	if err := sh.ix.insert(h, l); err == errRetry { //nolint:errorlint // errRetry is never wrapped.
		return loc{}, err
	} else if err != nil {
		return l, err
	}

	sh.count++

	return loc{}, nil
}

// Remove deletes key from the map. Removing a missing key is not an error.
func (m *Map) Remove(key string) error {
	h := m.hash(key, m.seed)
	sh := m.shardFor(h)

	var (
		checkBuf   [2]keyCheck
		frame, slot int
		old        loc
		found      bool
		err        error
	)

	checks := checkBuf[:0]

	sh.mu.Lock()

	for {
		if err = sh.usable(m.log); err == nil {
			frame, slot, old, found, err = sh.find(m.log, h, key, &checks)
		}

		if err != errRetry { //nolint:errorlint // errRetry is never wrapped.
			break
		}
	}

	if found {
		removeSlot(sh.pc.buf(frame), slot)
		sh.pc.markDirty(frame)
		sh.count--
	}

	sh.mu.Unlock()

	if found {
		m.log.markDead(old.seg, old.size)
	}

	return err
}

// Size returns the number of entries in the map.
func (m *Map) Size() int {
	total := 0

	for _, sh := range m.shards {
		sh.mu.Lock()
		total += sh.count
		sh.mu.Unlock()
	}

	return total
}

// Clear removes every entry and deletes all data files.
func (m *Map) Clear() error {
	// Take every shard and wait for its page writer to finish any write in
	// progress. Once a shard is locked and idle its writer cannot start
	// another, because that needs the lock.
	for _, sh := range m.shards {
		sh.mu.Lock()
		defer sh.mu.Unlock() //nolint:gocritic // Every shard stays locked until Clear returns.

		sh.pc.idle()
	}

	lg := m.log

	lg.mu.Lock()
	defer lg.mu.Unlock()

	if err := m.shards[0].usable(lg); err != nil {
		return err
	}

	// Write out the buffers rather than discarding them: a Get that found
	// its record before Clear took the locks may still be reading it.
	if err := lg.drain(); err != nil {
		return err
	}

	for _, s := range lg.table.Load().all() {
		s.drop()
	}

	lg.table.Store(&segTable{})
	lg.total, lg.dead = 0, 0
	lg.gen.Add(1)

	for _, sh := range m.shards {
		if err := sh.reset(); err != nil {
			return lg.fail(err)
		}
	}

	return lg.rotate()
}

// Close writes everything needed to reopen the map, syncs it to disk and
// releases the map's files. Closing a closed map is a no-op.
func (m *Map) Close() error {
	m.closeMu.Lock()
	defer m.closeMu.Unlock()

	if m.closed.Swap(true) {
		return nil
	}

	lg := m.log

	lg.mu.Lock()
	lg.closed = true
	err := lg.drain()
	lg.stopping = true
	lg.flushCond.Broadcast()
	lg.compactCond.Broadcast()
	lg.compactDone.Broadcast()
	lg.mu.Unlock()

	m.wg.Wait()

	err = errors.Join(err, m.persist())

	for _, sh := range m.shards {
		sh.mu.Lock()
		sh.pc.stop()
		sh.mu.Unlock()
	}

	m.writers.Wait()
	m.releaseFiles()

	return err
}

// persist writes back the index, syncs every file and writes the meta file.
// It marks every shard closed whether or not it succeeds.
func (m *Map) persist() error {
	saved := &meta{seed: m.seed}

	var errs []error

	for _, sh := range m.shards {
		sh.mu.Lock()

		err := sh.usable(m.log)
		sh.closed = true

		if err == nil {
			var sm shardMeta
			if sm, err = sh.persist(); err == nil {
				saved.shards = append(saved.shards, sm)
			}
		}

		if err != nil {
			errs = append(errs, err)
		}

		sh.mu.Unlock()
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	lg := m.log

	lg.mu.Lock()
	defer lg.mu.Unlock()

	saved.nextSeg = lg.nextSeg

	for _, s := range lg.table.Load().all() {
		if err := s.sync(); err != nil {
			return err
		}

		saved.segments = append(saved.segments, segmentMeta{id: s.id, size: s.size, dead: s.dead})
	}

	return writeMeta(m.dir, saved)
}

// releaseFiles drops the map's references to its files. Segment files close
// once in-flight reads finish.
func (m *Map) releaseFiles() {
	for _, s := range m.log.table.Load().all() {
		s.release()
	}

	for _, sh := range m.shards {
		_ = sh.idxFile.Close()
	}
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
	// IndexPages is the number of pages in the index files.
	IndexPages int
	// CachedPages is the number of index pages the caches can hold.
	CachedPages int
	// PageHits, PageMisses and PageWrites count index cache lookups served
	// from memory, lookups that read an index file, and dirty pages written
	// back.
	PageHits   uint64
	PageMisses uint64
	PageWrites uint64
	// CompactedBytes is the number of segment bytes compaction has processed.
	CompactedBytes int64
}

// Stats returns a snapshot of the map's statistics.
func (m *Map) Stats() Stats {
	var st Stats

	for _, sh := range m.shards {
		sh.mu.Lock()
		st.Entries += sh.count
		st.IndexPages += int(sh.ix.numPages)
		st.CachedPages += sh.pc.frames()
		st.PageHits += sh.pc.hits
		st.PageMisses += sh.pc.misses
		st.PageWrites += sh.pc.writes
		sh.mu.Unlock()
	}

	lg := m.log

	lg.mu.Lock()
	st.Segments = len(lg.table.Load().all())
	st.SegmentBytes = lg.total
	st.DeadBytes = lg.dead
	st.CompactedBytes = lg.compacted
	lg.mu.Unlock()

	return st
}
