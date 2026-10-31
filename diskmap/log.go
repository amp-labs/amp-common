package diskmap

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// durability says when a write may return.
type durability int

const (
	// durableWritten returns once the record is in the operating system,
	// so it survives the process crashing.
	durableWritten durability = iota
	// durableSynced returns once the record is on stable storage.
	durableSynced
	// durableBuffered returns at once; the record reaches the operating
	// system within bufferedFlushDelay.
	durableBuffered
)

const bufferedFlushDelay = 2 * time.Millisecond

// valueLog is the append-only store of records, shared by all shards. Its
// lock guards the write queue, sequence numbers and segment accounting;
// reads of records that are already in a file need no lock at all.
//
// Puts and tombstones are numbered by nextSeq and queued in memory buffers.
// A writer that must wait for its record to reach the file becomes the
// leader if no write is in progress: it writes every queued buffer, so
// concurrent writers share one system call (and, with synced durability, one
// fsync). Records still queued are served to readers from memory.
type valueLog struct {
	mu sync.Mutex

	dir      string
	segSize  int64
	mode     durability
	table    atomic.Pointer[segTable]
	nextSeg  uint32
	dirDirty bool // segment files were created since the directory was synced

	active     *segment // receives puts and tombstones
	compactOut *segment // receives moves; written only by the compactor
	retired    []retiredSegment

	nextSeq     uint64 // sequence number of the next put or tombstone
	nextMoveSeq uint64 // sequence number of the next move
	writtenSeq  uint64 // every put and tombstone below this is in a file
	syncedSeq   uint64 // every put and tombstone below this is synced

	queue     []*memBuf // buffers not yet written, oldest first
	inFlight  []*memBuf // buffers a leader is writing
	queued    int       // bytes in queue
	writing   bool
	syncing   bool
	bufCap    int
	spare     [][]byte
	writeCond *sync.Cond

	total  int64 // bytes in all segments
	dead   int64 // bytes of overwritten, removed or relocated records
	stalls int64 // times a writer waited for the disk

	// gen counts Clear calls. A compaction batch reserved in one
	// generation must not be indexed in another.
	gen atomic.Uint64

	// iterators counts Range calls in progress; compaction pauses while it is
	// nonzero.
	iterators atomic.Int32

	// writeMu is held for reading from the moment a record is appended until
	// the index points at it (by Add, Remove, and the compactor for each
	// batch it relocates), and for writing while Range or a checkpoint takes
	// its snapshot. A snapshot therefore never sees a record that is appended
	// but not yet indexed.
	writeMu sync.RWMutex

	// Compaction coordination; see compact.go.
	compactCond  *sync.Cond
	compactDone  *sync.Cond
	compactIdle  bool
	forceCompact bool
	compacted    int64
	threshold    float64

	// Checkpoint coordination; see checkpoint.go.
	checkpointGen   uint64 // incremented when a checkpoint freezes the map
	sinceCheckpoint int64  // bytes appended since the last checkpoint froze
	checkpointBytes int64
	checkpointCond  *sync.Cond
	checkpoints     int64

	closed   bool // no more appends
	stopping bool // background goroutines should exit
	failed   atomic.Pointer[error]
}

// memBuf holds records of one segment that are not yet in its file.
type memBuf struct {
	seg    *segment
	start  int64 // segment offset of data[0]
	data   []byte
	endSeq uint64 // sequence number after the last record in data
}

func (b *memBuf) end() int64 {
	return b.start + int64(len(b.data))
}

// retiredSegment is a compacted segment whose file must outlive the
// checkpoint that may still refer to it.
type retiredSegment struct {
	seg *segment
	gen uint64 // checkpointGen when it was retired
}

func newValueLog(dir string, segSize int64, bufCap int, threshold float64, mode durability, ckpt int64) *valueLog {
	lg := &valueLog{
		dir:             dir,
		segSize:         segSize,
		mode:            mode,
		bufCap:          bufCap,
		threshold:       threshold,
		checkpointBytes: ckpt,
	}
	lg.table.Store(&segTable{})
	lg.writeCond = sync.NewCond(&lg.mu)
	lg.compactCond = sync.NewCond(&lg.mu)
	lg.compactDone = sync.NewCond(&lg.mu)
	lg.checkpointCond = sync.NewCond(&lg.mu)

	return lg
}

func (lg *valueLog) segment(id uint32) *segment {
	return lg.table.Load().get(id)
}

func (lg *valueLog) err() error {
	if p := lg.failed.Load(); p != nil {
		return *p
	}

	return nil
}

// fail marks the map unusable after a failed write left its files
// inconsistent. The caller must hold lg.mu.
func (lg *valueLog) fail(err error) error {
	if lg.failed.CompareAndSwap(nil, &err) {
		lg.writeCond.Broadcast()
		lg.compactCond.Broadcast()
		lg.compactDone.Broadcast()
		lg.checkpointCond.Broadcast()
	}

	return err
}

// reserve waits, if too much is queued, for the backlog to be written. It
// must be called without a shard lock, before appending.
func (lg *valueLog) reserve() error {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	for lg.queued >= 2*lg.bufCap {
		err := lg.usable()
		if err != nil {
			return err
		}

		if !lg.writing {
			lg.writeQueued()

			continue
		}

		lg.stalls++
		lg.writeCond.Wait()
	}

	return lg.usable()
}

func (lg *valueLog) usable() error {
	err := lg.err()
	if err != nil {
		return err
	}

	if lg.closed {
		return ErrClosed
	}

	return nil
}

// appendRecord queues a put (or a tombstone) and returns its location and
// sequence number. It never waits; callers hold the key's shard lock, so
// appends and index updates of one key happen in the same order.
func (lg *valueLog) appendRecord(key string, value []byte, tombstone bool) (loc, uint64, error) {
	size := recordSize(len(key), len(value))

	lg.mu.Lock()
	defer lg.mu.Unlock()

	err := lg.usable()
	if err != nil {
		return loc{}, 0, err
	}

	if lg.active.size > 0 && lg.active.size+int64(size) > lg.segSize {
		err = lg.rotate()
		if err != nil {
			return loc{}, 0, err
		}
	}

	buf := lg.tail()
	seq := lg.nextSeq
	lg.nextSeq++

	addr := loc{seg: lg.active.id, off: uint32(lg.active.size), size: uint32(size)} //nolint:gosec // Below 4 GiB.
	buf.data = appendRecord(buf.data, seq, key, value, tombstone)
	buf.endSeq = seq + 1

	lg.active.size += int64(size)
	lg.total += int64(size)
	lg.queued += size
	lg.sinceCheckpoint += int64(size)

	if tombstone {
		lg.markDeadLocked(addr.seg, int64(size))
	}

	if lg.sinceCheckpoint >= lg.checkpointBytes {
		lg.checkpointCond.Signal()
	}

	return addr, seq, nil
}

// tail returns the queued buffer that new records for the active segment go
// into, starting one if needed.
func (lg *valueLog) tail() *memBuf {
	if n := len(lg.queue); n > 0 {
		if last := lg.queue[n-1]; last.seg == lg.active && len(last.data) < lg.bufCap {
			return last
		}
	}

	var data []byte

	if n := len(lg.spare); n > 0 {
		data = lg.spare[n-1]
		lg.spare = lg.spare[:n-1]
	} else {
		data = make([]byte, 0, lg.bufCap)
	}

	buf := &memBuf{seg: lg.active, start: lg.active.size, data: data}
	lg.queue = append(lg.queue, buf)

	if len(lg.queue) == 1 {
		lg.writeCond.Broadcast()
	}

	return buf
}

// rotate starts a new active segment.
func (lg *valueLog) rotate() error {
	seg, err := createSegment(lg.dir, lg.nextSeg, false)
	if err != nil {
		return lg.fail(err)
	}

	lg.nextSeg++
	lg.dirDirty = true
	lg.table.Store(lg.table.Load().with(seg))
	lg.active = seg
	lg.wakeCompactor(nil)

	return nil
}

// writeQueued writes every queued buffer, in order, with the lock released.
// The caller must hold lg.mu and no other write may be in progress.
func (lg *valueLog) writeQueued() {
	// After a failure the files may be inconsistent; write nothing more.
	if lg.err() != nil {
		lg.queue, lg.queued = nil, 0
		lg.writeCond.Broadcast()

		return
	}

	bufs := lg.queue
	lg.queue = nil
	lg.inFlight = bufs
	lg.writing = true

	err := lg.writeUnlocked(bufs)

	lg.writing = false
	lg.inFlight = nil

	if err != nil {
		_ = lg.fail(err)
	}

	for _, buf := range bufs {
		lg.queued -= len(buf.data)

		if err == nil {
			buf.seg.written.Store(buf.end())
			lg.writtenSeq = max(lg.writtenSeq, buf.endSeq)

			if buf.seg != lg.active {
				lg.wakeCompactor(buf.seg)
			}
		}

		if cap(buf.data) == lg.bufCap {
			lg.spare = append(lg.spare, buf.data[:0])
		}
	}

	lg.writeCond.Broadcast()
}

func (lg *valueLog) writeUnlocked(bufs []*memBuf) error {
	lg.mu.Unlock()
	defer lg.mu.Lock()

	for _, buf := range bufs {
		_, err := buf.seg.file.WriteAt(buf.data, buf.start)
		if err != nil {
			return fmt.Errorf("diskmap: writing segment %d: %w", buf.seg.id, err)
		}
	}

	return nil
}

// waitDurable waits until the put or tombstone numbered seq is as durable as
// the map's mode requires.
func (lg *valueLog) waitDurable(seq uint64) error {
	switch lg.mode {
	case durableBuffered:
		return nil
	case durableSynced:
		return lg.waitSynced(seq)
	case durableWritten:
		fallthrough
	default:
		lg.mu.Lock()
		defer lg.mu.Unlock()

		return lg.waitWritten(seq)
	}
}

// waitWritten waits until every put and tombstone up to seq is in a file,
// writing the queue itself if nobody else is. The caller must hold lg.mu.
func (lg *valueLog) waitWritten(seq uint64) error {
	for lg.writtenSeq <= seq {
		err := lg.err()
		if err != nil {
			return err
		}

		if !lg.writing && len(lg.queue) > 0 {
			lg.writeQueued()

			continue
		}

		lg.writeCond.Wait()
	}

	return nil
}

// waitSynced waits until every put and tombstone up to seq is on stable
// storage, along with everything else written before it.
func (lg *valueLog) waitSynced(seq uint64) error {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	for lg.syncedSeq <= seq {
		err := lg.waitWritten(seq)
		if err != nil {
			return err
		}

		if lg.syncing {
			lg.writeCond.Wait()

			continue
		}

		err = lg.syncWritten()
		if err != nil {
			return lg.fail(err)
		}
	}

	return nil
}

// syncWritten fsyncs every segment with written but unsynced bytes, and the
// directory if segments were created, with the lock released. Afterwards
// everything that was in a file when it started is on stable storage.
func (lg *valueLog) syncWritten() error {
	lg.syncing = true
	target := lg.writtenSeq
	dirDirty := lg.dirDirty
	lg.dirDirty = false

	type pending struct {
		seg     *segment
		written int64
	}

	var todo []pending

	for _, seg := range lg.table.Load().all() {
		if written := seg.written.Load(); seg.synced < written {
			seg.acquire()
			todo = append(todo, pending{seg: seg, written: written})
		}
	}

	lg.mu.Unlock()

	var err error

	for _, p := range todo {
		if err == nil {
			err = p.seg.file.Sync()
		}
	}

	if err == nil && dirDirty {
		err = syncDir(lg.dir)
	}

	lg.mu.Lock()

	for _, p := range todo {
		if err == nil {
			p.seg.synced = max(p.seg.synced, p.written)
		}

		p.seg.release()
	}

	lg.syncing = false
	lg.writeCond.Broadcast()

	if err != nil {
		lg.dirDirty = lg.dirDirty || dirDirty

		return fmt.Errorf("diskmap: syncing: %w", err)
	}

	lg.syncedSeq = max(lg.syncedSeq, target)

	return nil
}

// syncFrozen makes the records a checkpoint covers durable: everything
// numbered below saved.seq, in the segments it lists. Those segments are no
// longer written to, so this never delays a writer.
func (lg *valueLog) syncFrozen(saved *meta) error {
	lg.mu.Lock()

	if saved.seq > 0 {
		err := lg.waitWritten(saved.seq - 1)
		if err != nil {
			lg.mu.Unlock()

			return err
		}
	}

	var segs []*segment

	for _, sm := range saved.segments {
		if seg := lg.segment(sm.id); seg != nil && seg.synced < seg.written.Load() {
			seg.acquire()
			segs = append(segs, seg)
		}
	}

	dirDirty := lg.dirDirty
	lg.dirDirty = false
	lg.mu.Unlock()

	var err error

	for _, seg := range segs {
		written := seg.written.Load()

		if err == nil {
			err = seg.file.Sync()
		}

		if err == nil {
			lg.mu.Lock()
			seg.synced = max(seg.synced, written)
			lg.mu.Unlock()
		}

		seg.release()
	}

	if err == nil && dirDirty {
		err = syncDir(lg.dir)
	}

	if err != nil {
		lg.mu.Lock()
		lg.dirDirty = lg.dirDirty || dirDirty
		lg.mu.Unlock()

		return fmt.Errorf("diskmap: syncing: %w", err)
	}

	return nil
}

// syncAll makes everything appended so far durable: it writes the queue and
// fsyncs every segment.
func (lg *valueLog) syncAll() error {
	lg.mu.Lock()
	last := lg.nextSeq
	lg.mu.Unlock()

	if last == 0 {
		lg.mu.Lock()
		defer lg.mu.Unlock()

		for lg.syncing {
			lg.writeCond.Wait()
		}

		return lg.syncWritten()
	}

	return lg.waitSynced(last - 1)
}

// drain writes every queued record to its file.
func (lg *valueLog) drain() error {
	for {
		err := lg.err()
		if err != nil {
			return err
		}

		if lg.writing {
			lg.writeCond.Wait()

			continue
		}

		if len(lg.queue) == 0 {
			return nil
		}

		lg.writeQueued()
	}
}

// flushLoop writes queued records that no writer is waiting for, so
// buffered writes reach the file within bufferedFlushDelay.
func (lg *valueLog) flushLoop(wg *sync.WaitGroup) {
	defer wg.Done()

	lg.mu.Lock()
	defer lg.mu.Unlock()

	for {
		for len(lg.queue) == 0 || lg.writing {
			if lg.stopping {
				return
			}

			lg.writeCond.Wait()
		}

		if lg.mode == durableBuffered && lg.queued < lg.bufCap/2 {
			lg.mu.Unlock()
			time.Sleep(bufferedFlushDelay)
			lg.mu.Lock()
		}

		if !lg.writing && len(lg.queue) > 0 {
			lg.writeQueued()
		}
	}
}

// reserveMoves reserves space in the compaction output segment and sequence
// numbers for a batch of count moves totalling size bytes. at is the put
// sequence the batch follows. ok is false if compaction should stop.
func (lg *valueLog) reserveMoves(size, count int, gen uint64) (out *segment, base int64, first, at uint64, ok bool,
	err error,
) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	if lg.stopping || lg.closed || lg.gen.Load() != gen {
		return nil, 0, 0, 0, false, nil
	}

	err = lg.err()
	if err != nil {
		return nil, 0, 0, 0, false, err
	}

	out = lg.compactOut
	if out == nil || out.size > 0 && out.size+int64(size) > lg.segSize {
		out, err = createSegment(lg.dir, lg.nextSeg, true)
		if err != nil {
			return nil, 0, 0, 0, false, lg.fail(err)
		}

		lg.nextSeg++
		lg.dirDirty = true
		lg.table.Store(lg.table.Load().with(out))
		lg.compactOut = out
	}

	base = out.size
	out.size += int64(size)
	lg.total += int64(size)
	lg.sinceCheckpoint += int64(size)

	first = lg.nextMoveSeq
	lg.nextMoveSeq += uint64(count) //nolint:gosec // Non-negative.

	out.acquire()

	return out, base, first, lg.nextSeq, true, nil
}

// retire removes a compacted segment from the map. Its file is deleted once
// a later checkpoint is durable, since the current one may refer to it.
func (lg *valueLog) retire(seg *segment) {
	lg.table.Store(lg.table.Load().without(seg.id))
	lg.total -= seg.size
	lg.dead -= seg.dead
	lg.retired = append(lg.retired, retiredSegment{seg: seg, gen: lg.checkpointGen})
}

// markDead records that size bytes of segment id no longer hold a live
// record.
func (lg *valueLog) markDead(id uint32, size uint32) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	lg.markDeadLocked(id, int64(size))
}

func (lg *valueLog) markDeadLocked(id uint32, size int64) {
	s := lg.segment(id)
	if s == nil {
		return
	}

	s.dead += size
	lg.dead += size
	lg.wakeCompactor(s)
}

// read fills buf with the bytes of segment s starting at off. The caller
// must hold a reference to s or otherwise keep it from being dropped. Bytes
// already in the file are read without any lock; the newest bytes may still
// be queued in memory, which takes the log lock to copy.
func (lg *valueLog) read(s *segment, off int64, buf []byte) error {
	if off+int64(len(buf)) > s.written.Load() {
		lg.mu.Lock()
		buf = buf[:lg.copyBuffered(s, off, buf)]
		lg.mu.Unlock()
	}

	if len(buf) == 0 {
		return nil
	}

	return readFile(s, off, buf)
}

// copyBuffered copies the parts of [off, off+len(buf)) of segment s that are
// still in memory and returns how many leading bytes of buf must be read
// from the file instead. The caller must hold lg.mu.
func (lg *valueLog) copyBuffered(s *segment, off int64, buf []byte) int {
	end := off + int64(len(buf))

	for _, bufs := range [...][]*memBuf{lg.inFlight, lg.queue} {
		for _, b := range bufs {
			if b.seg != s {
				continue
			}

			if lo, hi := max(off, b.start), min(end, b.end()); lo < hi {
				copy(buf[lo-off:hi-off], b.data[lo-b.start:hi-b.start])
			}
		}
	}

	return int(min(max(s.written.Load()-off, 0), int64(len(buf))))
}
