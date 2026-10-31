package diskmap

import (
	"slices"
)

type recordState int

const (
	recordLive recordState = iota
	recordDead
	recordCleared
)

// span is the part of a segment that a Range visits.
type span struct {
	seg   *segment
	limit int64
}

// Range calls yield for each entry in the map, in roughly the order entries
// were written, until yield returns false. yield receives its own copy of
// each value.
//
// yield may read and modify the map. Entries added or overwritten during the
// iteration may or may not be visited, entries removed before they are
// reached are not visited, and no entry is visited twice. Compaction pauses
// while any Range is in progress, and Clear ends the iteration early.
func (m *Map) Range(yield func(key string, value []byte) bool) error {
	if m.closed.Load() {
		return ErrClosed
	}

	spans, gen, err := m.log.snapshot()
	if err != nil {
		return err
	}

	defer m.log.endIteration(spans)

	var reader chunkReader

	for _, sp := range spans {
		reader.n = 0

		for off := int64(0); off < sp.limit; {
			rec, err := reader.record(m.log, sp.seg, off, sp.limit)
			if err != nil {
				return err
			}

			state, err := m.recordState(sp.seg, off, rec, gen)
			if err != nil || state == recordCleared {
				return err
			}

			off += int64(len(rec))

			if state == recordDead {
				continue
			}

			key, value, err := parseRecord(rec, m.opts.verifyChecksums)
			if err != nil {
				return err
			}

			if !yield(string(key), slices.Clone(value)) {
				return nil
			}
		}
	}

	return nil
}

// recordState reports whether the record rec at (seg, off) is the current
// value of its key, or whether Clear has run since the iteration began.
func (m *Map) recordState(seg *segment, off int64, rec []byte, gen uint64) (recordState, error) {
	key := recordKey(rec)
	if key == nil {
		return recordCleared, ErrCorrupt
	}

	hash := m.hash(bytesToString(key), m.seed)
	shd := m.shardFor(hash)

	shd.mu.Lock()
	defer shd.mu.Unlock()

	err := shd.usable(m.log)
	if err != nil {
		return recordCleared, err
	}

	if m.log.gen.Load() != gen {
		return recordCleared, nil
	}

	_, live, err := shd.liveSlot(hash, seg.id, off)
	if err != nil || !live {
		return recordDead, err
	}

	return recordLive, nil
}

// snapshot records every segment and where it ends now, for a Range to
// visit, and pauses compaction. Records appended later are not visited.
// Holding writeMu means every record before the snapshot's end is already
// indexed.
func (lg *valueLog) snapshot() ([]span, uint64, error) {
	lg.writeMu.Lock()
	defer lg.writeMu.Unlock()

	lg.mu.Lock()
	defer lg.mu.Unlock()

	err := lg.err()
	if err != nil {
		return nil, 0, err
	}

	segs := lg.table.Load().all()
	spans := make([]span, 0, len(segs))

	for _, seg := range segs {
		seg.acquire()
		spans = append(spans, span{seg: seg, limit: seg.size})
	}

	lg.iterators.Add(1)

	return spans, lg.gen.Load(), nil
}

// endIteration undoes snapshot.
func (lg *valueLog) endIteration(spans []span) {
	lg.mu.Lock()

	if lg.iterators.Add(-1) == 0 {
		lg.compactCond.Broadcast()
	}

	lg.mu.Unlock()

	for _, sp := range spans {
		sp.seg.release()
	}
}
