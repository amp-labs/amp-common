package diskmap

import (
	"slices"
)

// Range calls fn for each entry in the map, in roughly the order entries were
// written, until fn returns false. fn receives its own copy of each value.
//
// fn may read and modify the map. Entries added or overwritten during the
// iteration may or may not be visited, entries removed before they are
// reached are not visited, and no entry is visited twice. Compaction pauses
// while any Range is in progress, and Clear ends the iteration early.
func (m *Map) Range(fn func(key string, value []byte) bool) error {
	m.mu.Lock()

	if err := m.usable(); err != nil {
		m.mu.Unlock()

		return err
	}

	it := &iterator{
		gen:    m.gen,
		endSeg: m.active.id,
		endOff: m.active.size,
	}
	ids := m.segmentIDs()
	m.iterators++

	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.iterators--
		m.mu.Unlock()
	}()

	for _, id := range ids {
		it.seg, it.off, it.reader.n = id, 0, 0

		for {
			key, value, state, err := m.rangeNext(it)
			if err != nil || state == iterCleared {
				return err
			}

			if state == iterSegmentDone {
				break
			}

			if state == iterLive && !fn(key, value) {
				return nil
			}
		}
	}

	return nil
}

type iterState int

const (
	iterLive iterState = iota
	iterDead
	iterSegmentDone
	iterCleared
)

type iterator struct {
	gen    uint64
	endSeg uint32
	endOff int64
	seg    uint32
	off    int64
	reader chunkReader
}

// rangeNext reads the next record of the iterator's segment and reports
// whether it is live.
func (m *Map) rangeNext(it *iterator) (string, []byte, iterState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.usable(); err != nil {
		return "", nil, 0, err
	}

	if m.gen != it.gen {
		return "", nil, iterCleared, nil
	}

	// Compaction is paused while iterating, so only Clear removes segments.
	s := m.segs[it.seg]
	if s == nil {
		return "", nil, iterSegmentDone, nil
	}

	limit := s.size
	if s.id == it.endSeg {
		limit = it.endOff
	}

	if it.off >= limit {
		return "", nil, iterSegmentDone, nil
	}

	rec, err := it.reader.record(m, s, it.off, limit)
	if err != nil {
		return "", nil, 0, err
	}

	off := it.off
	it.off += int64(len(rec))

	_, _, live, err := m.liveSlot(s.id, off, rec)
	if err != nil || !live {
		return "", nil, iterDead, err
	}

	key, value, err := parseRecord(rec, m.opts.verifyChecksums)
	if err != nil {
		return "", nil, 0, err
	}

	return string(key), slices.Clone(value), iterLive, nil
}
