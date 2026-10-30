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

	// Seal the active segment so its dead bytes can be reclaimed too.
	if vlog.cur.seg.dead > 0 {
		err := vlog.drain()
		if err != nil {
			return err
		}

		err = vlog.rotate()
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

// compactLoop is the compactor goroutine.
func (m *Map) compactLoop() {
	defer m.wg.Done()

	vlog := m.log

	var reader chunkReader

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	for {
		victim := vlog.nextVictim()
		if victim == nil {
			return
		}

		victim.acquire()

		size, gen := victim.size, vlog.gen.Load()
		done := m.compactUnlocked(victim, size, gen, &reader)

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
func (m *Map) compactUnlocked(victim *segment, size int64, gen uint64, reader *chunkReader) bool {
	m.log.mu.Unlock()
	defer m.log.mu.Lock()

	return m.compactVictim(victim, size, gen, reader)
}

// compactVictim relocates the victim's live records. It returns true once the
// whole victim has been processed, or false if it stopped early (stopping,
// Clear, or a failure). It is called without any lock held.
func (m *Map) compactVictim(victim *segment, size int64, gen uint64, reader *chunkReader) bool {
	vlog := m.log
	reader.n = 0

	for off := int64(0); off < size; {
		if !vlog.mayCompact(gen) {
			return false
		}

		vlog.writeMu.RLock()

		if vlog.iterators.Load() > 0 {
			vlog.writeMu.RUnlock()

			continue
		}

		var (
			moved  int64
			wasted []loc
			err    error
		)

		for batch := int64(0); off < size && batch < compactBatch; {
			var rec []byte

			rec, err = reader.record(vlog, victim, off, size)
			if err != nil {
				break
			}

			var (
				relocated bool
				copied    loc
			)

			relocated, copied, err = m.relocate(victim, off, rec, gen)
			if copied.size > 0 {
				wasted = append(wasted, copied)
			}

			if err != nil {
				break
			}

			if relocated {
				moved += int64(len(rec))
			}

			off += int64(len(rec))
			batch += int64(len(rec))
		}

		vlog.writeMu.RUnlock()

		if !vlog.settleBatch(victim, moved, wasted, err) {
			return false
		}
	}

	return true
}

// relocate copies the record at (victim, off) to the end of the log if the
// index still points at it, and repoints the index. relocated reports
// whether it did. If the record died while it was being copied, the copy is
// returned as wasted, to be counted as dead.
func (m *Map) relocate(victim *segment, off int64, rec []byte, gen uint64) (relocated bool, wasted loc, err error) {
	key := recordKey(rec)
	if key == nil {
		return false, loc{}, ErrCorrupt
	}

	hash := m.hash(bytesToString(key), m.seed)
	shd := m.shardFor(hash)

	shd.mu.Lock()
	_, live, err := shd.liveSlot(hash, victim.id, off)
	shd.mu.Unlock()

	if err != nil || !live {
		return false, loc{}, err
	}

	copied, copiedGen, err := m.log.appendRaw(rec)
	if err != nil {
		return false, loc{}, err
	}

	shd.mu.Lock()
	defer shd.mu.Unlock()

	if copiedGen != gen || shd.usable(m.log) != nil {
		return false, copied, nil
	}

	ref, live, err := shd.liveSlot(hash, victim.id, off)
	if err != nil || !live {
		return false, copied, err
	}

	setSlotLoc(shd.pc.buf(ref.frame), ref.slot, copied)
	shd.pc.markDirty(ref.frame)

	return true, loc{}, nil
}

// settleBatch records the outcome of a batch of relocations: relocated
// records are dead in the victim, and copies that lost a race with a
// concurrent write are dead where they landed. It reports whether compaction
// may continue.
func (lg *valueLog) settleBatch(victim *segment, moved int64, wasted []loc, err error) bool {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	victim.dead += moved
	lg.dead += moved
	lg.compacted += moved

	for _, copied := range wasted {
		lg.markDeadLocked(copied.seg, int64(copied.size))
	}

	if err != nil {
		_ = lg.fail(fmt.Errorf("diskmap: compacting segment %d: %w", victim.id, err))

		return false
	}

	return true
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
		if seg == lg.cur.seg || seg.dead == 0 {
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

	if seg != nil && (seg == lg.cur.seg || float64(seg.dead) < lg.threshold/victimShare*float64(seg.size)) {
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
