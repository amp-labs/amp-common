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
// A map lives in its own directory and survives crashes. By default every
// Add and Remove reaches the operating system before it returns, so it
// survives the process dying; Sync (or WithSyncWrites) makes writes survive
// a power failure too. Checkpoints taken in the background bound how much
// the next Open replays after a crash, and Close takes a final one.
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
	"sync/atomic"

	"github.com/zeebo/xxh3"
)

var (
	// ErrClosed is returned by operations on a closed map.
	ErrClosed = errors.New("diskmap: map is closed")

	// ErrLocked is returned by Open when another open map is using the
	// directory.
	ErrLocked = errors.New("diskmap: directory is in use")

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

	backgroundGoroutines = 3 // the flusher, the compactor and the checkpointer
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
	lock   *os.File

	closed       atomic.Bool
	closeMu      sync.Mutex
	checkpointMu sync.Mutex
	wg           sync.WaitGroup // flusher, compactor and checkpointer
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
	// WriteStalls counts the times a write waited for buffered data to reach
	// the disk because the disk had fallen behind.
	WriteStalls int64
	// Checkpoints counts checkpoints taken since Open.
	Checkpoints int64
}

// Open opens the map stored in dir, creating the directory and an empty map
// if needed. If the map was not closed, Open recovers it: it loads the last
// checkpoint and replays what was written after it. The map must be closed
// with Close.
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

	err = os.MkdirAll(dir, dirPerm)
	if err != nil {
		return nil, fmt.Errorf("diskmap: creating directory: %w", err)
	}

	lock, err := lockDir(dir)
	if err != nil {
		return nil, err
	}

	saved, err := prepareDir(dir, cfg.discardExisting)
	if err != nil {
		_ = lock.Close()

		return nil, err
	}

	if saved != nil {
		cfg.shards = len(saved.shards)
	}

	// Write buffers and the compactor's read and write buffers come out of
	// the budget; the rest holds index pages, split evenly between shards.
	bufCap := min(max(cfg.memoryLimit/writeBufferShare, minWriteBuffer), maxWriteBuffer)
	pages := (cfg.memoryLimit - cfg.valueCache - 2*bufCap - chunkSize - compactBatch) / pageSize / cfg.shards

	dmap := &Map{
		dir:  dir,
		opts: cfg,
		hash: cfg.hash,
		mask: uint64(cfg.shards - 1), //nolint:gosec // Validated positive.
		lock: lock,
		log: newValueLog(dir, cfg.segmentSize, bufCap, cfg.compactionThreshold, cfg.durability,
			cfg.checkpointBytes),
	}

	var garbage []string

	if saved == nil {
		err = dmap.create(pages)
	} else {
		garbage, err = dmap.recover(saved, pages)
	}

	if err != nil {
		dmap.releaseFiles()

		return nil, err
	}

	dmap.start()

	// A new map needs its first checkpoint, and a recovered one a fresh one
	// so the replayed records are not replayed again.
	err = dmap.checkpoint(false)
	if err != nil {
		dmap.shutdown()

		return nil, err
	}

	for _, name := range garbage {
		_ = os.Remove(filepath.Join(dir, name))
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

	return m.write(key, value, false)
}

// Remove deletes key from the map. Removing a missing key is not an error.
func (m *Map) Remove(key string) error {
	return m.write(key, nil, true)
}

// write appends a put or tombstone and points the index at it, both under
// the key's shard lock, so a key's records are numbered in the order they
// took effect and recovery replays them that way. It then waits, without
// any lock, until the record is as durable as the map requires.
func (m *Map) write(key string, value []byte, tombstone bool) error {
	if m.closed.Load() {
		return ErrClosed
	}

	hash := m.hash(key, m.seed)
	shd := m.shardFor(hash)
	vlog := m.log

	err := vlog.reserve()
	if err != nil {
		return err
	}

	vlog.writeMu.RLock()
	shd.mu.Lock()

	seq, wrote, err := m.writeLocked(shd, hash, key, value, tombstone)

	shd.mu.Unlock()
	vlog.writeMu.RUnlock()

	if err != nil || !wrote {
		return err
	}

	return vlog.waitDurable(seq)
}

// writeLocked does the work of write under the shard lock. It returns the
// record's sequence number and whether it appended anything (removing a
// missing key does not).
func (m *Map) writeLocked(
	shd *shard, hash uint64, key string, value []byte, tombstone bool,
) (seq uint64, wrote bool, err error) {
	var checkBuf [2]keyCheck

	checks := checkBuf[:0]

	for {
		err = shd.usable(m.log)
		if err != nil {
			return 0, false, err
		}

		var (
			ref   slotRef
			found bool
		)

		ref, found, err = shd.find(m.log, hash, key, &checks)
		if errors.Is(err, errRetry) {
			continue
		}

		if err != nil || tombstone && !found {
			return 0, false, err
		}

		var addr loc

		addr, seq, err = m.log.appendRecord(key, value, tombstone)
		if err != nil {
			return 0, false, err
		}

		return seq, true, m.applyFound(shd, hash, ref, found, addr, tombstone)
	}
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

	vlog := m.log

	vlog.writeMu.Lock()
	defer vlog.writeMu.Unlock()

	old, err := m.clearLocked()
	if err != nil {
		return err
	}

	// The old files stay until the empty map's checkpoint is durable, since
	// until then the previous checkpoint refers to them.
	err = m.checkpoint(true)
	if err != nil {
		return err
	}

	for _, seg := range old {
		seg.drop()
	}

	return nil
}

// clearLocked empties the index and the log and returns the segments to
// delete. The caller holds writeMu for writing.
func (m *Map) clearLocked() ([]*segment, error) {
	// Take every shard and wait for its page writer to finish any write in
	// progress. Once a shard is locked and idle its writer cannot start
	// another, because that needs the lock.
	for _, shd := range m.shards {
		shd.mu.Lock()
		defer shd.mu.Unlock() //nolint:gocritic // Every shard stays locked until the index is reset.

		shd.pc.idle()
	}

	vlog := m.log

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	err := vlog.usable()
	if err != nil {
		return nil, err
	}

	// Write out the queue rather than discarding it: a Get that found its
	// record before Clear took the locks may still be reading it.
	err = vlog.drain()
	if err != nil {
		return nil, err
	}

	old := vlog.table.Load().all()
	for _, r := range vlog.retired {
		old = append(old, r.seg)
	}

	vlog.retired = nil
	vlog.table.Store(&segTable{})
	vlog.compactOut = nil
	vlog.total, vlog.dead = 0, 0
	vlog.gen.Add(1)

	for _, shd := range m.shards {
		err = shd.reset()
		if err != nil {
			return nil, vlog.fail(err)
		}
	}

	return old, vlog.rotate()
}

// Close takes a final checkpoint, syncs everything to disk and releases the
// map's files. Closing a closed map is a no-op.
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
	vlog.writeCond.Broadcast()
	vlog.compactCond.Broadcast()
	vlog.compactDone.Broadcast()
	vlog.checkpointCond.Broadcast()
	vlog.mu.Unlock()

	m.wg.Wait()

	if err == nil {
		err = m.checkpoint(false)
	}

	m.shutdown()

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
	stats.WriteStalls = vlog.stalls
	stats.Checkpoints = vlog.checkpoints
	vlog.mu.Unlock()

	return stats
}

// prepareDir creates dir if needed and loads the last checkpoint of the map
// in it, returning nil for a new map.
func prepareDir(dir string, discard bool) (*meta, error) {
	if discard {
		err := removeDataFiles(dir)
		if err != nil {
			return nil, err
		}
	}

	saved, err := readMeta(dir)
	if err != nil || saved != nil {
		return saved, err
	}

	// Without a checkpoint the map never finished opening, so it holds
	// nothing a caller was told was written.
	return nil, removeDataFiles(dir)
}

// create starts an empty map.
func (m *Map) create(pages int) error {
	var seed [8]byte

	_, _ = rand.Read(seed[:])
	m.seed = binary.LittleEndian.Uint64(seed[:])

	for i := range m.opts.shards {
		shd, err := openShard(m.dir, i, pages, m.opts.valueCache/m.opts.shards)
		if err != nil {
			return err
		}

		m.shards = append(m.shards, shd)

		err = shd.create()
		if err != nil {
			return err
		}
	}

	m.log.mu.Lock()
	defer m.log.mu.Unlock()

	return m.log.rotate()
}

// start launches the background goroutines.
func (m *Map) start() {
	m.wg.Add(backgroundGoroutines)

	go m.log.flushLoop(&m.wg)
	go m.compactLoop()
	go m.checkpointLoop()

	for _, shd := range m.shards {
		m.writers.Add(1)

		go shd.pc.writeLoop(&m.writers)
	}
}

// shutdown stops the background goroutines and releases the map's files.
func (m *Map) shutdown() {
	vlog := m.log

	vlog.mu.Lock()
	vlog.closed, vlog.stopping = true, true
	vlog.writeCond.Broadcast()
	vlog.compactCond.Broadcast()
	vlog.compactDone.Broadcast()
	vlog.checkpointCond.Broadcast()
	vlog.mu.Unlock()

	m.wg.Wait()

	for _, shd := range m.shards {
		shd.mu.Lock()
		shd.closed = true
		shd.pc.stop()
		shd.mu.Unlock()
	}

	m.writers.Wait()
	m.releaseFiles()
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

// releaseFiles drops the map's references to its files. Segment files close
// once in-flight reads finish.
func (m *Map) releaseFiles() {
	for _, seg := range m.log.table.Load().all() {
		seg.release()
	}

	for _, r := range m.log.retired {
		r.seg.release()
	}

	m.log.retired = nil

	if m.lock != nil {
		_ = m.lock.Close()
	}

	for _, shd := range m.shards {
		_ = shd.idxFile.Close()
	}
}
