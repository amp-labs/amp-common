// Package diskmap provides a disk-backed map from string keys to byte-slice
// values, for maps that have outgrown memory.
//
// Values are appended to segment files, and an on-disk hash index, read
// through a fixed-size page cache, says where each one is. Memory use stays
// near the configured limit no matter how many entries the map holds.
//
// Every lookup reads exactly one index page (usually from the cache) and one
// record. The index is split into independently locked shards, and file
// reads and writes happen outside every lock, so goroutines rarely wait on
// each other. Growth splits one index page at a time, and background
// goroutines flush writes and compact dead space in small steps, so no
// single operation pays for a rehash, a flush or a compaction pass.
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
	// writeBufferShare is the fraction of the memory limit, as a divisor,
	// given to each of the two write buffers, between minWriteBuffer and
	// maxWriteBuffer.
	writeBufferShare = 32
	maxWriteBuffer   = 1 << 20
	minWriteBuffer   = 64 << 10
	chunkSize        = 64 << 10

	backgroundGoroutines = 2 // the flusher and the compactor
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
	// CompactedBytes is the number of live record bytes compaction has moved.
	CompactedBytes int64
	// ValueCacheHits and ValueCacheMisses count reads served from the value
	// cache and reads that had to go to a segment file. Both stay zero
	// without WithValueCache.
	ValueCacheHits   uint64
	ValueCacheMisses uint64
}

// Open opens the map stored in dir, creating the directory and an empty map
// if needed. The map must be closed with Close.
func Open(dir string, opts ...Option) (*Map, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	err := cfg.validate()
	if err != nil {
		return nil, err
	}

	if cfg.hash == nil {
		cfg.hash = xxh3.HashStringSeed
	}

	saved, err := prepareDir(dir, cfg.discardExisting)
	if err != nil {
		return nil, err
	}

	if saved != nil {
		cfg.shards = len(saved.shards)
	}

	// Two write buffers and the compactor's chunk come out of the budget;
	// the rest holds index pages, split evenly between shards.
	bufCap := min(max(cfg.memoryLimit/writeBufferShare, minWriteBuffer), maxWriteBuffer)
	pages := (cfg.memoryLimit - cfg.valueCache - 2*bufCap - chunkSize) / pageSize / cfg.shards

	dmap := &Map{
		dir:  dir,
		opts: cfg,
		hash: cfg.hash,
		mask: uint64(cfg.shards - 1), //nolint:gosec // Validated positive.
		log:  newValueLog(dir, cfg.segmentSize, bufCap, cfg.compactionThreshold),
	}

	err = dmap.open(saved, pages)
	if err != nil {
		dmap.releaseFiles()

		return nil, err
	}

	dmap.wg.Add(backgroundGoroutines)

	go dmap.log.flushLoop(&dmap.wg)
	go dmap.compactLoop()

	for _, shd := range dmap.shards {
		dmap.writers.Add(1)

		go shd.pc.writeLoop(&dmap.writers)
	}

	return dmap, nil
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

// Contains reports whether key is in the map.
func (m *Map) Contains(key string) (bool, error) {
	hash := m.hash(key, m.seed)
	shd := m.shardFor(hash)

	var checkBuf [2]keyCheck

	checks := checkBuf[:0]

	shd.mu.Lock()
	defer shd.mu.Unlock()

	for {
		err := shd.usable(m.log)
		if err != nil {
			return false, err
		}

		_, found, err := shd.find(m.log, hash, key, &checks)
		if !errors.Is(err, errRetry) {
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

	hash := m.hash(key, m.seed)
	shd := m.shardFor(hash)

	// Append first, then point the index at the new record. Concurrent Adds
	// of one key may append in one order and index in the other; either
	// result is a valid outcome of the race.
	m.log.writeMu.RLock()

	addr, gen, err := m.log.appendRecord(key, value)
	if err != nil {
		m.log.writeMu.RUnlock()

		return err
	}

	var checkBuf [2]keyCheck

	checks := checkBuf[:0]

	shd.mu.Lock()

	dead, err := m.indexRecord(shd, hash, key, addr, gen, &checks)
	for errors.Is(err, errRetry) {
		dead, err = m.indexRecord(shd, hash, key, addr, gen, &checks)
	}

	shd.mu.Unlock()
	m.log.writeMu.RUnlock()

	if dead.size > 0 {
		m.log.markDead(dead.seg, dead.size)
	}

	return err
}

// Remove deletes key from the map. Removing a missing key is not an error.
func (m *Map) Remove(key string) error {
	hash := m.hash(key, m.seed)
	shd := m.shardFor(hash)

	var (
		checkBuf [2]keyCheck
		ref      slotRef
		found    bool
		err      error
	)

	checks := checkBuf[:0]

	shd.mu.Lock()

	for {
		err = shd.usable(m.log)
		if err == nil {
			ref, found, err = shd.find(m.log, hash, key, &checks)
		}

		if !errors.Is(err, errRetry) {
			break
		}
	}

	if found {
		removeSlot(shd.pc.buf(ref.frame), ref.slot)
		shd.pc.markDirty(ref.frame)
		shd.count--
	}

	shd.mu.Unlock()

	if found {
		m.log.markDead(ref.addr.seg, ref.addr.size)
	}

	return err
}

// Size returns the number of entries in the map.
func (m *Map) Size() int {
	total := 0

	for _, shd := range m.shards {
		shd.mu.Lock()
		total += shd.count
		shd.mu.Unlock()
	}

	return total
}

// Clear removes every entry and deletes all data files.
func (m *Map) Clear() error {
	if m.closed.Load() {
		return ErrClosed
	}

	// Take every shard and wait for its page writer to finish any write in
	// progress. Once a shard is locked and idle its writer cannot start
	// another, because that needs the lock.
	for _, shd := range m.shards {
		shd.mu.Lock()
		defer shd.mu.Unlock() //nolint:gocritic // Every shard stays locked until Clear returns.

		shd.pc.idle()
	}

	vlog := m.log

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	if vlog.closed {
		return ErrClosed
	}

	err := m.shards[0].usable(vlog)
	if err != nil {
		return err
	}

	// Write out the buffers rather than discarding them: a Get that found
	// its record before Clear took the locks may still be reading it.
	err = vlog.drain()
	if err != nil {
		return err
	}

	for _, seg := range vlog.table.Load().all() {
		seg.drop()
	}

	vlog.table.Store(&segTable{})
	vlog.total, vlog.dead = 0, 0
	vlog.gen.Add(1)

	for _, shd := range m.shards {
		err = shd.reset()
		if err != nil {
			return vlog.fail(err)
		}
	}

	return vlog.rotate()
}

// Close writes everything needed to reopen the map, syncs it to disk and
// releases the map's files. Closing a closed map is a no-op.
func (m *Map) Close() error {
	m.closeMu.Lock()
	defer m.closeMu.Unlock()

	if m.closed.Swap(true) {
		return nil
	}

	vlog := m.log

	vlog.mu.Lock()
	vlog.closed = true
	err := vlog.drain()
	vlog.stopping = true
	vlog.flushCond.Broadcast()
	vlog.compactCond.Broadcast()
	vlog.compactDone.Broadcast()
	vlog.mu.Unlock()

	m.wg.Wait()

	err = errors.Join(err, m.persist())

	for _, shd := range m.shards {
		shd.mu.Lock()
		shd.pc.stop()
		shd.mu.Unlock()
	}

	m.writers.Wait()
	m.releaseFiles()

	return err
}

// Stats returns a snapshot of the map's statistics.
func (m *Map) Stats() Stats {
	var stats Stats

	for _, shd := range m.shards {
		shd.mu.Lock()
		stats.Entries += shd.count
		stats.IndexPages += int(shd.ix.numPages)
		stats.CachedPages += shd.pc.frames()
		stats.PageHits += shd.pc.hits
		stats.PageMisses += shd.pc.misses
		stats.PageWrites += shd.pc.writes

		if shd.vc != nil {
			stats.ValueCacheHits += shd.vc.hits
			stats.ValueCacheMisses += shd.vc.misses
		}

		shd.mu.Unlock()
	}

	vlog := m.log

	vlog.mu.Lock()
	stats.Segments = len(vlog.table.Load().all())
	stats.SegmentBytes = vlog.total
	stats.DeadBytes = vlog.dead
	stats.CompactedBytes = vlog.compacted
	vlog.mu.Unlock()

	return stats
}

// prepareDir creates dir if needed and loads the saved state of the map in
// it, returning nil for a new map.
func prepareDir(dir string, discard bool) (*meta, error) {
	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return nil, fmt.Errorf("diskmap: creating directory: %w", err)
	}

	if discard {
		err = removeDataFiles(dir)
		if err != nil {
			return nil, err
		}
	}

	saved, err := readMeta(dir)
	if err != nil || saved != nil {
		return saved, err
	}

	files, err := dataFiles(dir)
	if err != nil {
		return nil, err
	}

	if len(files) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrNotCleanlyClosed, dir)
	}

	return nil, nil
}

func (m *Map) open(saved *meta, pages int) error {
	for i := range m.opts.shards {
		shd, err := openShard(m.dir, i, pages, m.opts.valueCache/m.opts.shards)
		if err != nil {
			return err
		}

		m.shards = append(m.shards, shd)
	}

	if saved == nil {
		var seed [8]byte

		_, _ = rand.Read(seed[:])
		m.seed = binary.LittleEndian.Uint64(seed[:])

		for _, shd := range m.shards {
			err := shd.create()
			if err != nil {
				return err
			}
		}

		return m.log.rotate()
	}

	m.seed = saved.seed

	for i, shd := range m.shards {
		err := shd.load(saved.shards[i])
		if err != nil {
			return err
		}
	}

	err := m.log.load(saved)
	if err != nil {
		return err
	}

	return removeMeta(m.dir)
}

func (m *Map) shardFor(hash uint64) *shard {
	return m.shards[hash&m.mask]
}

// get appends the value for key to dst; a nil dst gets an exactly sized
// fresh slice. The index lookup happens under the shard lock and the record
// read outside it.
func (m *Map) get(dst []byte, key string) ([]byte, bool, error) {
	hash := m.hash(key, m.seed)
	shd := m.shardFor(hash)

	shd.mu.Lock()

	var (
		addr    loc
		matches int
		err     error
	)

	for {
		err = shd.usable(m.log)
		if err == nil {
			addr, matches, err = shd.candidates(hash)
		}

		if !errors.Is(err, errRetry) {
			break
		}
	}

	if err != nil || matches == 0 {
		shd.mu.Unlock()

		return dst, false, err
	}

	if matches > 1 {
		// Distinct keys with equal 64-bit hashes: rare enough to handle the
		// simple way, entirely under the lock (the page is cached now).
		defer shd.mu.Unlock()

		return m.getCollided(shd, dst, key, hash)
	}

	admit := false

	if shd.vc != nil {
		out, found, cached := cachedValue(shd.vc, dst, addr, key)
		if cached {
			shd.mu.Unlock()

			return out, found, nil
		}

		admit = shd.vc.admit(addr, int(addr.size))
	}

	// The index points into seg, so it cannot be dropped while the shard is
	// locked; the reference keeps it open after the lock is released.
	seg := m.log.segment(addr.seg)
	if seg == nil {
		shd.mu.Unlock()

		return dst, false, fmt.Errorf("%w: index points to missing segment %d", ErrCorrupt, addr.seg)
	}

	seg.acquire()
	shd.mu.Unlock()

	defer seg.release()

	if dst == nil {
		dst = make([]byte, 0, addr.size)
	}

	start := len(dst)
	grown := slices.Grow(dst, int(addr.size))
	buf := grown[start : start+int(addr.size)]

	err = m.log.read(seg, int64(addr.off), buf)
	if err != nil {
		return dst, false, err
	}

	n, ok, err := m.extractValue(buf, key, addr)
	if err != nil || !ok {
		return dst, false, err
	}

	if admit {
		shd.mu.Lock()
		shd.vc.put(addr, key, buf[:n])
		shd.mu.Unlock()
	}

	return grown[:start+n], true, nil
}

// cachedValue appends the value of the record at addr to dst if the value
// cache holds that record. cached reports whether it did; found reports
// whether the record belongs to key.
func cachedValue(vc *valueCache, dst []byte, addr loc, key string) (out []byte, found, cached bool) {
	value, cached := vc.get(addr, key)
	if !cached || value == nil {
		return dst, false, cached
	}

	if dst == nil {
		dst = make([]byte, 0, len(value))
	}

	return append(dst, value...), true, true
}

// getCollided is get for a key whose hash matches several slots. It runs
// under the shard lock.
func (m *Map) getCollided(shd *shard, dst []byte, key string, hash uint64) ([]byte, bool, error) {
	frame, err := shd.ix.frameFor(hash)
	if err != nil {
		return dst, false, err
	}

	page := shd.pc.buf(frame)
	mt := matchSlots(page, hash)

	for slot := mt.next(); slot >= 0; slot = mt.next() {
		addr := slotLoc(page, slot)

		seg := m.log.segment(addr.seg)
		if seg == nil {
			return dst, false, fmt.Errorf("%w: index points to missing segment %d", ErrCorrupt, addr.seg)
		}

		buf := make([]byte, addr.size)

		err = m.log.read(seg, int64(addr.off), buf)
		if err != nil {
			return dst, false, err
		}

		n, ok, err := m.extractValue(buf, key, addr)
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
func (m *Map) extractValue(rec []byte, key string, addr loc) (int, bool, error) {
	recKey, value, err := parseRecord(rec, m.opts.verifyChecksums)
	if err != nil {
		return 0, false, fmt.Errorf("record at segment %d offset %d: %w", addr.seg, addr.off, err)
	}

	if string(recKey) != key {
		return 0, false, nil
	}

	return copy(rec, value), true, nil
}

// indexRecord points key at the record addr, appended in generation gen, and
// returns the location of the record that no longer holds a live value: the
// one addr replaced, or addr itself if it could not be indexed. It returns
// errRetry if it had to release the lock.
func (m *Map) indexRecord(
	shd *shard, hash uint64, key string, addr loc, gen uint64, checks *[]keyCheck,
) (loc, error) {
	err := shd.usable(m.log)
	if err != nil {
		return addr, err
	}

	// Clear ran after the append and deleted the record with everything else.
	if m.log.gen.Load() != gen {
		return loc{}, nil
	}

	ref, found, err := shd.find(m.log, hash, key, checks)
	if errors.Is(err, errRetry) {
		return loc{}, err
	}

	if err != nil {
		return addr, err
	}

	if found {
		setSlotLoc(shd.pc.buf(ref.frame), ref.slot, addr)
		shd.pc.markDirty(ref.frame)

		return ref.addr, nil
	}

	err = shd.ix.insert(hash, addr)
	if errors.Is(err, errRetry) {
		return loc{}, err
	}

	if err != nil {
		return addr, err
	}

	shd.count++

	return loc{}, nil
}

// persist writes back the index, syncs every file and writes the meta file.
// It marks every shard closed whether or not it succeeds.
func (m *Map) persist() error {
	saved := &meta{seed: m.seed}

	var errs []error

	for _, shd := range m.shards {
		shd.mu.Lock()

		err := shd.usable(m.log)
		shd.closed = true

		if err == nil {
			var state shardMeta

			state, err = shd.persist()
			if err == nil {
				saved.shards = append(saved.shards, state)
			}
		}

		if err != nil {
			errs = append(errs, err)
		}

		shd.mu.Unlock()
	}

	err := errors.Join(errs...)
	if err != nil {
		return err
	}

	vlog := m.log

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	saved.nextSeg = vlog.nextSeg

	for _, seg := range vlog.table.Load().all() {
		err = seg.sync()
		if err != nil {
			return err
		}

		saved.segments = append(saved.segments, segmentMeta{id: seg.id, size: seg.size, dead: seg.dead})
	}

	return writeMeta(m.dir, saved)
}

// releaseFiles drops the map's references to its files. Segment files close
// once in-flight reads finish.
func (m *Map) releaseFiles() {
	for _, seg := range m.log.table.Load().all() {
		seg.release()
	}

	for _, shd := range m.shards {
		_ = shd.idxFile.Close()
	}
}
