package diskmap

import (
	"fmt"
	"math"
	"unsafe"
)

const (
	// compactionFactor is how many victim bytes compaction processes per byte
	// written or removed. A victim is at least half dead (see compactStep), so
	// this reclaims space at least twice as fast as writes create garbage.
	compactionFactor = 4

	// minCompactionStep keeps small writes from compacting a few bytes at a
	// time.
	minCompactionStep = 4 << 10
)

// compactor tracks an incremental compaction of one victim segment.
type compactor struct {
	victim    *segment
	off       int64
	reader    chunkReader
	processed int64
}

func (c *compactor) reset() {
	c.victim = nil
	c.off = 0
	c.reader.n = 0
}

// compactStep does a slice of compaction work proportional to the bytes an
// operation just wrote or killed, so space is reclaimed steadily instead of
// in long pauses. It picks a new victim only once dead bytes exceed the
// threshold, and only a segment that is itself well past half the threshold.
func (m *Map) compactStep(written int64) error {
	threshold := m.opts.compactionThreshold
	if threshold == 0 || m.iterators > 0 {
		return nil
	}

	if m.comp.victim == nil {
		if float64(m.dead) <= threshold*float64(m.total) {
			return nil
		}

		if m.comp.victim = m.pickVictim(threshold / 2); m.comp.victim == nil { //nolint:mnd // Half the threshold.
			return nil
		}
	}

	return m.compact(max(written*compactionFactor, minCompactionStep))
}

// Compact reclaims all space held by overwritten and removed entries. It
// blocks other operations until it finishes, and does nothing while a Range
// is in progress.
func (m *Map) Compact() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.usable(); err != nil {
		return err
	}

	if m.iterators > 0 {
		return nil
	}

	if m.active.dead > 0 {
		if err := m.rotate(); err != nil {
			return err
		}
	}

	for {
		if m.comp.victim == nil {
			if m.comp.victim = m.pickVictim(0); m.comp.victim == nil {
				return nil
			}
		}

		if err := m.compact(math.MaxInt64); err != nil {
			return err
		}
	}
}

// pickVictim returns the sealed segment with the largest fraction of dead
// bytes, if that fraction is at least minRatio.
func (m *Map) pickVictim(minRatio float64) *segment {
	var best *segment

	bestRatio := minRatio

	for _, s := range m.segs {
		if s == m.active || s.dead == 0 {
			continue
		}

		if ratio := float64(s.dead) / float64(s.size); ratio >= bestRatio {
			best, bestRatio = s, ratio
		}
	}

	return best
}

// compact processes up to budget bytes of the victim, copying live records to
// the active segment, and deletes the victim once it has been fully read.
func (m *Map) compact(budget int64) error {
	victim := m.comp.victim

	for budget > 0 && m.comp.off < victim.size {
		rec, err := m.comp.reader.record(m, victim, m.comp.off, victim.size)
		if err != nil {
			return m.fail(fmt.Errorf("diskmap: compacting segment %d: %w", victim.id, err))
		}

		if err := m.relocate(victim.id, m.comp.off, rec); err != nil {
			return m.fail(err)
		}

		size := int64(len(rec))
		m.comp.off += size
		m.comp.processed += size
		budget -= size
	}

	if m.comp.off < victim.size {
		return nil
	}

	m.comp.reset()
	delete(m.segs, victim.id)
	m.total -= victim.size
	m.dead -= victim.dead

	if err := victim.remove(); err != nil {
		return m.fail(err)
	}

	return nil
}

// relocate copies the record at (seg, off) to the active segment if the index
// still points at it.
func (m *Map) relocate(seg uint32, off int64, rec []byte) error {
	frame, slot, live, err := m.liveSlot(seg, off, rec)
	if err != nil || !live {
		return err
	}

	l, err := m.appendRaw(rec)
	if err != nil {
		return err
	}

	// Appending touches only segment files, so frame still holds the page.
	setSlotLoc(m.pc.buf(frame), slot, l)
	m.pc.markDirty(frame)

	return nil
}

// liveSlot finds the index slot pointing at the record rec stored at
// (seg, off). The record is live exactly when such a slot exists.
func (m *Map) liveSlot(seg uint32, off int64, rec []byte) (frame, slot int, live bool, err error) {
	keyLen, _ := recordLengths(rec)
	if recordHeaderSize+keyLen > len(rec) {
		return 0, 0, false, ErrCorrupt
	}

	h := m.hash(bytesToString(rec[recordHeaderSize:recordHeaderSize+keyLen]), m.seed)

	frame, err = m.ix.frameFor(h)
	if err != nil {
		return 0, 0, false, err
	}

	p := m.pc.buf(frame)

	for i := range pageCount(p) {
		if slotHash(p, i) != h {
			continue
		}

		if l := slotLoc(p, i); l.seg == seg && int64(l.off) == off {
			return frame, i, true, nil
		}
	}

	return frame, 0, false, nil
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

// record returns the record at off in segment s, whose data ends at limit.
// The slice is valid until the next call.
func (r *chunkReader) record(m *Map, s *segment, off, limit int64) ([]byte, error) {
	if r.buf == nil {
		r.buf = make([]byte, chunkSize)
	}

	if limit-off < recordHeaderSize {
		return nil, ErrCorrupt
	}

	if !r.has(off, recordHeaderSize) {
		if err := r.load(m, s, off, limit); err != nil {
			return nil, err
		}
	}

	keyLen, valueLen := recordLengths(r.buf[off-r.off:])
	size := int64(recordSize(keyLen, valueLen))

	if off+size > limit {
		return nil, ErrCorrupt
	}

	if size > int64(len(r.buf)) {
		rec := make([]byte, size)

		return rec, m.readAt(s, off, rec)
	}

	if !r.has(off, size) {
		if err := r.load(m, s, off, limit); err != nil {
			return nil, err
		}
	}

	return r.buf[off-r.off : off-r.off+size], nil
}

func (r *chunkReader) has(off, n int64) bool {
	return off >= r.off && off+n <= r.off+int64(r.n)
}

func (r *chunkReader) load(m *Map, s *segment, off, limit int64) error {
	n := min(int64(len(r.buf)), limit-off)

	r.n = 0
	if err := m.readAt(s, off, r.buf[:n]); err != nil {
		return err
	}

	r.off, r.n = off, int(n)

	return nil
}
