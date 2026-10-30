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
	if m.closed.Load() {
		return ErrClosed
	}

	lg := m.log

	// Snapshot the segments and where each one ends now; records appended
	// later are not visited. Holding writeMu means every record before the
	// snapshot's end is already indexed.
	type span struct {
		seg   *segment
		limit int64
	}

	lg.writeMu.Lock()
	lg.mu.Lock()

	if err := lg.err(); err != nil {
		lg.mu.Unlock()
		lg.writeMu.Unlock()

		return err
	}

	segs := lg.table.Load().all()
	spans := make([]span, 0, len(segs))

	for _, s := range segs {
		s.acquire()
		spans = append(spans, span{seg: s, limit: s.size})
	}

	gen := lg.gen.Load()
	lg.iterators.Add(1)

	lg.mu.Unlock()
	lg.writeMu.Unlock()

	defer func() {
		lg.mu.Lock()

		if lg.iterators.Add(-1) == 0 {
			lg.compactCond.Broadcast()
		}

		lg.mu.Unlock()

		for _, sp := range spans {
			sp.seg.release()
		}
	}()

	var reader chunkReader

	for _, sp := range spans {
		reader.n = 0

		for off := int64(0); off < sp.limit; {
			rec, err := reader.record(lg, sp.seg, off, sp.limit)
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

			if !fn(string(key), slices.Clone(value)) {
				return nil
			}
		}
	}

	return nil
}

type recordState int

const (
	recordLive recordState = iota
	recordDead
	recordCleared
)

// recordState reports whether the record rec at (s, off) is the current
// value of its key, or whether Clear has run since the iteration began.
func (m *Map) recordState(s *segment, off int64, rec []byte, gen uint64) (recordState, error) {
	keyLen, _ := recordLengths(rec)
	if recordHeaderSize+keyLen > len(rec) {
		return recordCleared, ErrCorrupt
	}

	h := m.hash(bytesToString(rec[recordHeaderSize:recordHeaderSize+keyLen]), m.seed)
	sh := m.shardFor(h)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if err := sh.usable(m.log); err != nil {
		return recordCleared, err
	}

	if m.log.gen.Load() != gen {
		return recordCleared, nil
	}

	_, _, live, err := sh.liveSlotRetry(h, s.id, off)
	if err != nil || !live {
		return recordDead, err
	}

	return recordLive, nil
}
