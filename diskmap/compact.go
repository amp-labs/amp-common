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

	if vlog.active.dead > 0 {
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
	out    []byte // the batch encoded as move records
}

// move is a live record copied into a compaction batch.
type move struct {
	hash uint64
	from loc    // where the record is in the victim
	at   int64  // offset of the record in the batch
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
			vlog.retire(victim)
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

		// Holding writeMu keeps a Range or checkpoint from starting between
		// copying a batch and repointing the index at it.
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
// off, into comp.batch and returns the offset after the batch. Tombstones
// are never live: the checkpoint that follows the victim's retirement covers
// them, and the file outlives it until then.
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

		live, hash, err := m.isLive(victim, off, rec)
		if err != nil {
			return off, err
		}

		if live {
			comp.moves = append(comp.moves, move{
				hash: hash,
				from: loc{seg: victim.id, off: uint32(off), size: uint32(len(rec))}, //nolint:gosec // Below 4 GiB.
				at:   int64(len(comp.batch)),
			})
			comp.batch = append(comp.batch, rec...)
		}

		off += int64(len(rec))
	}

	return off, nil
}

// isLive reports whether the record rec at (seg, off) is the current value
// of its key, and returns the key's hash.
func (m *Map) isLive(seg *segment, off int64, rec []byte) (bool, uint64, error) {
	if recordKindOf(rec) == kindTombstone {
		return false, 0, nil
	}

	key := recordKey(rec)
	if key == nil {
		return false, 0, ErrCorrupt
	}

	hash := m.hash(bytesToString(key), m.seed)
	shd := m.shardFor(hash)

	shd.mu.Lock()
	_, live, err := shd.liveSlot(hash, seg.id, off)
	shd.mu.Unlock()

	return live, hash, err
}

// relocateBatch writes comp.batch to the compaction output segment as move
// records and repoints the index at the copies. The copies are in the file
// before any slot points at them, so readers never need the log lock to
// find them. It reports stop if compaction should end because the map is
// closing or was cleared.
func (m *Map) relocateBatch(victim *segment, gen uint64, comp *compactor) (stop bool, err error) {
	vlog := m.log

	// A move record is its original plus a trailer, which a record that was
	// already a move has.
	size := 0

	for _, mv := range comp.moves {
		size += int(mv.from.size)
		if recordKindOf(comp.batch[mv.at:]) != kindMove {
			size += moveTrailerSize
		}
	}

	out, base, first, at, ok, err := vlog.reserveMoves(size, len(comp.moves), gen)
	if err != nil || !ok {
		return true, err
	}

	defer out.release()

	comp.out = comp.out[:0]
	copies := make([]loc, len(comp.moves))

	for i, mv := range comp.moves {
		start := len(comp.out)
		rec := comp.batch[mv.at : mv.at+int64(mv.from.size)]
		comp.out = appendMove(comp.out, rec, first+uint64(i), moveInfo{at: at, from: mv.from}) //nolint:gosec // Small.
		copies[i] = loc{seg: out.id, off: uint32(base) + uint32(start), size: uint32(len(comp.out) - start)} //nolint:gosec,lll // Below 4 GiB.
	}

	_, err = out.file.WriteAt(comp.out, base)
	if err != nil {
		return true, fmt.Errorf("diskmap: writing segment %d: %w", out.id, err)
	}

	out.written.Store(base + int64(len(comp.out)))

	var moved, wasted int64

	for i, mv := range comp.moves {
		repointed, err := m.repoint(victim, gen, mv, copies[i])
		if err != nil {
			return true, err
		}

		if repointed {
			moved += int64(mv.from.size)
		} else {
			wasted += int64(copies[i].size)
		}
	}

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	// Clear dropped both segments and reset the accounting meanwhile.
	if vlog.gen.Load() != gen {
		return true, nil
	}

	// Copies that lost a race with a concurrent write are dead where they
	// landed; relocated records are dead in the victim.
	out.dead += wasted
	victim.dead += moved
	vlog.dead += moved + wasted
	vlog.compacted += moved

	return false, nil
}

// repoint moves the index slot for a relocated record to its copy, if the
// slot still points at the original. It reports whether it did.
func (m *Map) repoint(victim *segment, gen uint64, mv move, copied loc) (bool, error) {
	shd := m.shardFor(mv.hash)

	shd.mu.Lock()
	defer shd.mu.Unlock()

	if shd.usable(m.log) != nil || m.log.gen.Load() != gen {
		return false, nil
	}

	ref, live, err := shd.liveSlot(mv.hash, victim.id, int64(mv.from.off))
	if err != nil || !live {
		return false, err
	}

	err = shd.pc.modify(ref.frame)
	if err != nil {
		return false, err
	}

	setSlotLoc(shd.pc.buf(ref.frame), ref.slot, copied)

	return true, nil
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
		if seg == lg.active || seg == lg.compactOut || seg.dead == 0 || seg.written.Load() < seg.size {
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

	if seg != nil && (seg == lg.active || seg == lg.compactOut ||
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

	size := int64(fullRecordSize(r.buf[off-r.off:]))

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

	if size := int64(fullRecordSize(r.buf)); size > length && off+size <= limit {
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
