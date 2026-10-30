package diskmap

import (
	"fmt"
	"unsafe"
)

// compactBatch bounds how many victim bytes the compactor processes between
// checks for stopping, Clear and new iterators.
const compactBatch = 64 << 10

// Compaction runs in a background goroutine. Once dead bytes exceed the
// threshold it picks the sealed segment with the largest dead fraction and
// reads it in chunks without holding any lock. For each record it asks the
// record's shard whether the index still points at it; live records are
// appended to the log again and the index repointed, each step taking its
// lock only briefly. When the whole segment has been read, it is dropped and
// its file deleted.

// compactLoop is the compactor goroutine.
func (m *Map) compactLoop() {
	defer m.wg.Done()

	lg := m.log

	var reader chunkReader

	lg.mu.Lock()
	defer lg.mu.Unlock()

	for {
		victim := lg.nextVictim()
		if victim == nil {
			return
		}

		victim.acquire()
		size, gen := victim.size, lg.gen.Load()

		lg.mu.Unlock()
		done := m.compactVictim(victim, size, gen, &reader)
		lg.mu.Lock()

		victim.release()

		if done && lg.gen.Load() == gen {
			lg.table.Store(lg.table.Load().without(victim.id))
			lg.total -= victim.size
			lg.dead -= victim.dead

			lg.mu.Unlock()
			victim.drop()
			lg.mu.Lock()
		}
	}
}

// nextVictim waits until there is a segment worth compacting and returns it,
// or returns nil when the map is stopping. The caller must hold lg.mu.
func (lg *valueLog) nextVictim() *segment {
	for {
		if lg.stopping || lg.closed {
			return nil
		}

		if lg.err() == nil && lg.iterators.Load() == 0 {
			if victim := lg.pickVictim(); victim != nil {
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
	minRatio := lg.threshold / 2 //nolint:mnd // Half the threshold.

	switch {
	case lg.forceCompact:
		minRatio = 0
	case lg.threshold == 0 || !lg.overThreshold():
		return nil
	}

	var best *segment

	bestRatio := minRatio

	for _, s := range lg.table.Load().all() {
		if s == lg.cur.seg || s.dead == 0 {
			continue
		}

		if ratio := float64(s.dead) / float64(s.size); ratio >= bestRatio {
			best, bestRatio = s, ratio
		}
	}

	return best
}

func (lg *valueLog) overThreshold() bool {
	return float64(lg.dead) > lg.threshold*float64(lg.total)
}

// wakeCompactor wakes an idle compactor if s, a segment that just gained dead
// bytes (or nil after a rotation), may now be worth compacting. The caller
// must hold lg.mu.
func (lg *valueLog) wakeCompactor(s *segment) {
	if !lg.compactIdle || lg.threshold == 0 || !lg.overThreshold() {
		return
	}

	if s != nil && (s == lg.cur.seg || float64(s.dead) < lg.threshold/2*float64(s.size)) {
		return
	}

	lg.compactIdle = false
	lg.compactCond.Signal()
}

// compactVictim relocates the victim's live records. It returns true once the
// whole victim has been processed, or false if it stopped early (stopping,
// Clear, or a failure). It is called without any lock held.
func (m *Map) compactVictim(victim *segment, size int64, gen uint64, reader *chunkReader) bool {
	lg := m.log
	reader.n = 0

	for off := int64(0); off < size; {
		if !lg.mayCompact(gen) {
			return false
		}

		lg.writeMu.RLock()

		if lg.iterators.Load() > 0 {
			lg.writeMu.RUnlock()

			continue
		}

		var (
			moved  int64
			wasted []loc
			err    error
		)

		for batch := int64(0); off < size && batch < compactBatch; {
			var rec []byte

			if rec, err = reader.record(lg, victim, off, size); err != nil {
				break
			}

			var copied loc

			if copied, err = m.relocate(victim, off, rec, gen); err != nil {
				break
			}

			switch {
			case copied.size == 0:
			case copied.seg == noSegment:
				moved += int64(len(rec))
			default:
				wasted = append(wasted, copied)
			}

			off += int64(len(rec))
			batch += int64(len(rec))
		}

		lg.writeMu.RUnlock()

		lg.mu.Lock()

		// Relocated records are dead in the victim; copies that lost a race
		// with a concurrent write are dead where they landed.
		victim.dead += moved
		lg.dead += moved
		lg.compacted += moved

		for _, l := range wasted {
			lg.markDeadLocked(l.seg, int64(l.size))
		}

		if err != nil {
			lg.fail(fmt.Errorf("diskmap: compacting segment %d: %w", victim.id, err))
		}

		lg.mu.Unlock()

		if err != nil {
			return false
		}
	}

	return true
}

// noSegment marks a relocation that moved the record: relocate returns a loc
// with this segment and the record's size.
const noSegment = ^uint32(0)

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

// relocate copies the record at (victim, off) to the end of the log if the
// index still points at it, and repoints the index. It returns a zero loc if
// the record was dead, {noSegment, size} if it moved, or the location of the
// copy if the record died while being copied, leaving the copy dead.
func (m *Map) relocate(victim *segment, off int64, rec []byte, gen uint64) (loc, error) {
	keyLen, _ := recordLengths(rec)
	if recordHeaderSize+keyLen > len(rec) {
		return loc{}, ErrCorrupt
	}

	h := m.hash(bytesToString(rec[recordHeaderSize:recordHeaderSize+keyLen]), m.seed)
	sh := m.shardFor(h)

	sh.mu.Lock()
	_, _, live, err := sh.liveSlotRetry(h, victim.id, off)
	sh.mu.Unlock()

	if err != nil || !live {
		return loc{}, err
	}

	copied, copiedGen, err := m.log.appendRaw(rec)
	if err != nil {
		return loc{}, err
	}

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if copiedGen != gen || sh.usable(m.log) != nil {
		return copied, nil
	}

	frame, slot, live, err := sh.liveSlotRetry(h, victim.id, off)
	if err != nil || !live {
		return copied, err
	}

	setSlotLoc(sh.pc.buf(frame), slot, copied)
	sh.pc.markDirty(frame)

	return loc{seg: noSegment, size: copied.size}, nil
}

// Compact reclaims all space held by overwritten and removed entries and
// waits for it to finish. Other operations continue meanwhile. Compact does
// nothing while a Range is in progress.
func (m *Map) Compact() error {
	if m.closed.Load() {
		return ErrClosed
	}

	lg := m.log

	lg.mu.Lock()
	defer lg.mu.Unlock()

	if lg.iterators.Load() > 0 {
		return nil
	}

	// Seal the active segment so its dead bytes can be reclaimed too.
	if lg.cur.seg.dead > 0 {
		if err := lg.drain(); err != nil {
			return err
		}

		if err := lg.rotate(); err != nil {
			return err
		}
	}

	lg.forceCompact = true
	lg.compactIdle = false
	lg.compactCond.Signal()

	for lg.forceCompact && !lg.stopping && lg.err() == nil {
		lg.compactDone.Wait()
	}

	if lg.stopping {
		return ErrClosed
	}

	return lg.err()
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

// record returns the record at off in segment s, whose data ends at limit,
// reading from the file as needed. The caller must hold a reference to s.
// The slice is valid until the next call.
func (r *chunkReader) record(lg *valueLog, s *segment, off, limit int64) ([]byte, error) {
	if rec, ok := r.buffered(off, limit); ok {
		return rec, nil
	}

	if err := r.load(lg, s, off, limit); err != nil {
		return nil, err
	}

	if rec, ok := r.buffered(off, limit); ok {
		return rec, nil
	}

	return nil, ErrCorrupt
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
func (r *chunkReader) load(lg *valueLog, s *segment, off, limit int64) error {
	// A record larger than a chunk grows the buffer; shrink it back so one
	// huge value does not pin memory.
	if len(r.buf) != chunkSize {
		r.buf = make([]byte, chunkSize)
	}

	r.n = 0

	n := min(int64(chunkSize), limit-off)
	if n < recordHeaderSize {
		return ErrCorrupt
	}

	if err := lg.read(s, off, r.buf[:n]); err != nil {
		return err
	}

	keyLen, valueLen := recordLengths(r.buf)
	if size := int64(recordSize(keyLen, valueLen)); size > n && off+size <= limit {
		r.buf = make([]byte, size)
		n = size

		if err := lg.read(s, off, r.buf[:n]); err != nil {
			return err
		}
	}

	r.off, r.n = off, int(n)

	return nil
}
