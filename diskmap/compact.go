package diskmap

import (
	"fmt"
	"unsafe"
)

const (
	// compactBatch bounds how many victim bytes the compactor processes
	// between checks for stopping, Clear and new iterators.
	compactBatch = 64 << 10

	// A segment qualifies for compaction once its dead fraction is at least
	// the threshold divided by victimShare.
	victimShare = 2
)

// Compaction runs in a background goroutine. Once dead bytes exceed the
// threshold it picks the sealed segment with the largest dead fraction and
// reads it in chunks without holding any lock. For each record it asks the
// record's shard whether the index still points at it; live records are
// appended to the log again and the index repointed, each step taking its
// lock only briefly. When the whole segment has been read, it is dropped and
// its file deleted.

// Compact reclaims all space held by overwritten and removed entries and
// waits for it to finish. Other operations continue meanwhile. Compact does
// nothing while a Range is in progress.
func (m *Map) Compact() error {
	if m.closed.Load() {
		return ErrClosed
	}

	vlog := m.log

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	if vlog.iterators.Load() > 0 {
		return nil
	}

	// Seal the segments being written and get every byte to disk, so all
	// dead bytes can be reclaimed.
	vlog.compactOut = nil

	err := vlog.drain()
	if err != nil {
		return err
	}

	if vlog.cur.seg.dead > 0 {
		err = vlog.rotate()
		if err != nil {
			return err
		}

		err = vlog.drain()
		if err != nil {
			return err
		}
	}

	vlog.forceCompact = true
	vlog.compactIdle = false
	vlog.compactCond.Signal()

	for vlog.forceCompact && !vlog.stopping && vlog.err() == nil {
		vlog.compactDone.Wait()
	}

	if vlog.stopping {
		return ErrClosed
	}

	return vlog.err()
}

// compactor holds the compaction goroutine's buffers.
type compactor struct {
	reader chunkReader
	batch  []byte // live records copied out of the victim
	moves  []move // where each record in batch came from
}

// move is a live record copied into a compaction batch.
type move struct {
	hash uint64
	from int64 // offset in the victim
	at   int64 // offset in the batch
	size uint32
}

// compactLoop is the compactor goroutine.
func (m *Map) compactLoop() {
	defer m.wg.Done()

	vlog := m.log

	var comp compactor

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	for {
		victim := vlog.nextVictim()
		if victim == nil {
			return
		}

		victim.acquire()

		size, gen := victim.size, vlog.gen.Load()
		done := m.compactUnlocked(victim, size, gen, &comp)

		victim.release()

		if done && vlog.gen.Load() == gen {
			vlog.table.Store(vlog.table.Load().without(victim.id))
			vlog.total -= victim.size
			vlog.dead -= victim.dead

			vlog.dropUnlocked(victim)
		}
	}
}

// compactUnlocked runs compactVictim with the log lock released.
func (m *Map) compactUnlocked(victim *segment, size int64, gen uint64, comp *compactor) bool {
	m.log.mu.Unlock()
	defer m.log.mu.Lock()

	return m.compactVictim(victim, size, gen, comp)
}

// compactVictim relocates the victim's live records, a batch at a time. It
// returns true once the whole victim has been processed, or false if it
// stopped early (stopping, Clear, or a failure). It is called without any
// lock held.
func (m *Map) compactVictim(victim *segment, size int64, gen uint64, comp *compactor) bool {
	vlog := m.log
	comp.reader.n = 0

	for off := int64(0); off < size; {
		if !vlog.mayCompact(gen) {
			return false
		}

		// Holding writeMu keeps a Range from starting between copying a
		// batch and repointing the index at it.
		vlog.writeMu.RLock()

		if vlog.iterators.Load() > 0 {
			vlog.writeMu.RUnlock()

			continue
		}

		next, err := m.gatherLive(victim, off, size, comp)

		stop := false
		if err == nil && len(comp.moves) > 0 {
			stop, err = m.relocateBatch(victim, gen, comp)
		}

		vlog.writeMu.RUnlock()

		if err != nil {
			vlog.mu.Lock()
			_ = vlog.fail(fmt.Errorf("diskmap: compacting segment %d: %w", victim.id, err))
			vlog.mu.Unlock()
		}

		if err != nil || stop {
			return false
		}

		off = next
	}

	return true
}

// gatherLive copies the live records of a batch of the victim, starting at
// off, into comp.batch and returns the offset after the batch.
func (m *Map) gatherLive(victim *segment, off, size int64, comp *compactor) (int64, error) {
	// A huge record leaves an outsized batch buffer; let it go.
	if cap(comp.batch) > 2*compactBatch {
		comp.batch = nil
	}

	comp.batch = comp.batch[:0]
	comp.moves = comp.moves[:0]

	for off < size && len(comp.batch) < compactBatch {
		rec, err := comp.reader.record(m.log, victim, off, size)
		if err != nil {
			return off, err
		}

		key := recordKey(rec)
		if key == nil {
			return off, ErrCorrupt
		}

		hash := m.hash(bytesToString(key), m.seed)
		shd := m.shardFor(hash)

		shd.mu.Lock()
		_, live, err := shd.liveSlot(hash, victim.id, off)
		shd.mu.Unlock()

		if err != nil {
			return off, err
		}

		if live {
			comp.moves = append(comp.moves, move{
				hash: hash,
				from: off,
				at:   int64(len(comp.batch)),
				size: uint32(len(rec)), //nolint:gosec // Records fit in uint32.
			})
			comp.batch = append(comp.batch, rec...)
		}

		off += int64(len(rec))
	}

	return off, nil
}

// relocateBatch writes comp.batch to the compaction output segment and
// repoints the index at the copies. The copies are in the file before any
// slot points at them, so readers never need the log lock to find them, and
// the compactor takes the log lock only twice per batch. It reports stop if
// compaction should end because the map is closing or was cleared.
func (m *Map) relocateBatch(victim *segment, gen uint64, comp *compactor) (stop bool, err error) {
	vlog := m.log

	out, base, ok, err := vlog.compactionOutput(len(comp.batch), gen)
	if err != nil || !ok {
		return true, err
	}

	defer out.release()

	_, err = out.file.WriteAt(comp.batch, base)
	if err != nil {
		return true, fmt.Errorf("diskmap: writing segment %d: %w", out.id, err)
	}

	out.written.Store(base + int64(len(comp.batch)))

	var moved, wasted int64

	for _, mv := range comp.moves {
		if m.repoint(victim, gen, mv, loc{seg: out.id, off: uint32(base + mv.at), size: mv.size}) { //nolint:gosec,lll // Segments stay below 4 GiB.
			moved += int64(mv.size)
		} else {
			wasted += int64(mv.size)
		}
	}

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	// Clear dropped both segments and reset the accounting meanwhile.
	if vlog.gen.Load() != gen {
		return true, nil
	}

	// The copies are counted whether or not they were used; those that lost
	// a race with a concurrent write are dead where they landed. Relocated
	// records are dead in the victim.
	out.size = base + int64(len(comp.batch))
	out.dead += wasted
	victim.dead += moved
	vlog.total += int64(len(comp.batch))
	vlog.dead += moved + wasted
	vlog.compacted += moved

	return false, nil
}

// repoint moves the index slot for a relocated record to its copy, if the
// slot still points at the original. It reports whether it did.
func (m *Map) repoint(victim *segment, gen uint64, mv move, copied loc) bool {
	shd := m.shardFor(mv.hash)

	shd.mu.Lock()
	defer shd.mu.Unlock()

	if shd.usable(m.log) != nil || m.log.gen.Load() != gen {
		return false
	}

	ref, live, err := shd.liveSlot(mv.hash, victim.id, mv.from)
	if err != nil || !live {
		return false
	}

	setSlotLoc(shd.pc.buf(ref.frame), ref.slot, copied)
	shd.pc.markDirty(ref.frame)

	return true
}

// compactionOutput returns the segment relocated records go to and the
// offset where n more bytes will land, starting a new segment when the
// current one is full. Relocated records are long-lived, so keeping them
// apart from new writes makes later compactions cheaper. ok is false if
// compaction should stop. The caller must release the returned segment.
func (lg *valueLog) compactionOutput(n int, gen uint64) (out *segment, base int64, ok bool, err error) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	if lg.stopping || lg.closed || lg.gen.Load() != gen {
		return nil, 0, false, nil
	}

	err = lg.err()
	if err != nil {
		return nil, 0, false, err
	}

	out = lg.compactOut
	if out == nil || out.size > 0 && out.size+int64(n) > lg.segSize {
		out, err = createSegment(lg.dir, lg.nextSeg)
		if err != nil {
			return nil, 0, false, lg.fail(err)
		}

		lg.nextSeg++
		lg.table.Store(lg.table.Load().with(out))
		lg.compactOut = out
	}

	out.acquire()
	out.unsynced = true

	return out, out.size, true, nil
}

// dropUnlocked deletes a dropped segment's file with the log lock released.
func (lg *valueLog) dropUnlocked(seg *segment) {
	lg.mu.Unlock()
	defer lg.mu.Lock()

	seg.drop()
}

// nextVictim waits until there is a segment worth compacting and returns it,
// or returns nil when the map is stopping. The caller must hold lg.mu.
func (lg *valueLog) nextVictim() *segment {
	for {
		if lg.stopping || lg.closed {
			return nil
		}

		if lg.err() == nil && lg.iterators.Load() == 0 {
			victim := lg.pickVictim()
			if victim != nil {
				lg.compactIdle = false

				return victim
			}

			if lg.forceCompact {
				lg.forceCompact = false
				lg.compactDone.Broadcast()
			}
		}

		lg.compactIdle = true
		lg.compactCond.Wait()
	}
}

// pickVictim returns the sealed segment with the largest fraction of dead
// bytes, if compaction is due. Normally that is when dead bytes exceed the
// threshold, and only segments at least half as dead as the threshold
// qualify; Compact makes every segment with dead bytes qualify.
func (lg *valueLog) pickVictim() *segment {
	minRatio := lg.threshold / victimShare

	switch {
	case lg.forceCompact:
		minRatio = 0
	case lg.threshold == 0 || !lg.overThreshold():
		return nil
	}

	var best *segment

	bestRatio := minRatio

	for _, seg := range lg.table.Load().all() {
		// A sealed segment whose last buffer is still being flushed is not
		// ready: dropping it would close the file under the flusher.
		if seg == lg.cur.seg || seg == lg.compactOut || seg.dead == 0 || seg.written.Load() < seg.size {
			continue
		}

		ratio := float64(seg.dead) / float64(seg.size)
		if ratio >= bestRatio {
			best, bestRatio = seg, ratio
		}
	}

	return best
}

func (lg *valueLog) overThreshold() bool {
	return float64(lg.dead) > lg.threshold*float64(lg.total)
}

// wakeCompactor wakes an idle compactor if seg, a segment that just gained
// dead bytes (or nil after a rotation), may now be worth compacting. The
// caller must hold lg.mu.
func (lg *valueLog) wakeCompactor(seg *segment) {
	if !lg.compactIdle || lg.threshold == 0 || !lg.overThreshold() {
		return
	}

	if seg != nil && (seg == lg.cur.seg || seg == lg.compactOut ||
		float64(seg.dead) < lg.threshold/victimShare*float64(seg.size)) {
		return
	}

	lg.compactIdle = false
	lg.compactCond.Signal()
}

// mayCompact waits while iterators are running and reports whether
// compaction of a victim from generation gen may continue.
func (lg *valueLog) mayCompact(gen uint64) bool {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	for lg.iterators.Load() > 0 && !lg.stopping && !lg.closed {
		lg.compactCond.Wait()
	}

	return !lg.stopping && !lg.closed && lg.err() == nil && lg.gen.Load() == gen
}

// bytesToString views b as a string without copying. The caller must not
// modify b while the string is in use.
func bytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b)) //nolint:gosec // Read-only view for hashing.
}

// chunkReader reads records from a segment through a reusable buffer, so
// sequential scans issue one read per chunk rather than one per record.
type chunkReader struct {
	buf []byte
	off int64 // segment offset of buf[0]
	n   int   // valid bytes in buf
}

// record returns the record at off in segment seg, whose data ends at limit,
// reading from the file as needed. The caller must hold a reference to seg.
// The slice is valid until the next call.
func (r *chunkReader) record(vlog *valueLog, seg *segment, off, limit int64) ([]byte, error) {
	rec, ok := r.buffered(off, limit)
	if ok {
		return rec, nil
	}

	err := r.load(vlog, seg, off, limit)
	if err != nil {
		return nil, err
	}

	rec, ok = r.buffered(off, limit)
	if !ok {
		return nil, ErrCorrupt
	}

	return rec, nil
}

// buffered returns the record at off if it is entirely in the buffer.
func (r *chunkReader) buffered(off, limit int64) ([]byte, bool) {
	if !r.has(off, recordHeaderSize) {
		return nil, false
	}

	keyLen, valueLen := recordLengths(r.buf[off-r.off:])
	size := int64(recordSize(keyLen, valueLen))

	if off+size > limit || !r.has(off, size) {
		return nil, false
	}

	return r.buf[off-r.off : off-r.off+size], true
}

func (r *chunkReader) has(off, n int64) bool {
	return off >= r.off && off+n <= r.off+int64(r.n)
}

// load fills the buffer starting at off, growing it to hold the record there
// if that record is larger than a chunk.
func (r *chunkReader) load(vlog *valueLog, seg *segment, off, limit int64) error {
	// A record larger than a chunk grows the buffer; shrink it back so one
	// huge value does not pin memory.
	if len(r.buf) != chunkSize {
		r.buf = make([]byte, chunkSize)
	}

	r.n = 0

	length := min(int64(chunkSize), limit-off)
	if length < recordHeaderSize {
		return ErrCorrupt
	}

	err := vlog.read(seg, off, r.buf[:length])
	if err != nil {
		return err
	}

	keyLen, valueLen := recordLengths(r.buf)
	if size := int64(recordSize(keyLen, valueLen)); size > length && off+size <= limit {
		r.buf = make([]byte, size)
		length = size

		err = vlog.read(seg, off, r.buf[:length])
		if err != nil {
			return err
		}
	}

	r.off, r.n = off, int(length)

	return nil
}
