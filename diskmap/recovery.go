package diskmap

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// stream walks the records written to one kind of segment after a
// checkpoint: puts and tombstones, or moves. Each segment's records are in
// sequence order, and so are the segments.
type stream struct {
	segs   []*segment
	starts []int64

	i      int   // current segment
	off    int64 // offset of the current record
	rec    []byte
	ok     bool
	reader chunkReader
}

// advance moves to the next complete, intact record. At the end of the
// stream, or at the first damaged record, ok becomes false and (i, off) is
// where the stream's valid data ends.
func (st *stream) advance(vlog *valueLog) {
	if st.ok {
		st.off += int64(len(st.rec))
	}

	st.ok = false

	for st.i < len(st.segs) {
		seg := st.segs[st.i]
		if st.off >= seg.size {
			if st.i+1 == len(st.segs) {
				return
			}

			st.i++
			st.off = st.starts[st.i]
			st.reader.n = 0

			continue
		}

		rec, err := st.reader.record(vlog, seg, st.off, seg.size)
		if err != nil || !validRecord(rec) {
			return
		}

		st.rec, st.ok = rec, true

		return
	}
}

func (st *stream) addr() loc {
	return loc{seg: st.segs[st.i].id, off: uint32(st.off), size: uint32(len(st.rec))} //nolint:gosec // Below 4 GiB.
}

// recover restores the map from the checkpoint saved and the records written
// after it. It returns the files no checkpoint will need, to delete once a
// new checkpoint is durable.
func (m *Map) recover(saved *meta, pages int) ([]string, error) {
	m.seed = saved.seed

	for i := range m.opts.shards {
		shd, err := openShard(m.dir, i, pages, m.opts.valueCache/m.opts.shards)
		if err != nil {
			return nil, err
		}

		m.shards = append(m.shards, shd)

		err = shd.loadCheckpoint(saved.shards[i])
		if err != nil {
			return nil, err
		}
	}

	vlog := m.log
	vlog.nextSeq, vlog.nextMoveSeq, vlog.nextSeg = saved.seq, saved.moveSeq, saved.nextSeg

	puts, moves, garbage, err := m.openSegments(saved)
	if err != nil {
		return nil, err
	}

	err = m.replay(saved, puts, moves)
	if err != nil {
		return nil, err
	}

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	vlog.writtenSeq = vlog.nextSeq

	return garbage, vlog.rotate()
}

// openSegments opens the segments the checkpoint lists and those created
// after it, and returns the two streams to replay and the names of segment
// files nothing refers to.
func (m *Map) openSegments(saved *meta) (puts, moves *stream, garbage []string, err error) {
	vlog := m.log
	listed := make(map[uint32]segmentMeta, len(saved.segments))

	for _, sm := range saved.segments {
		listed[sm.id] = sm
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("diskmap: reading directory: %w", err)
	}

	puts, moves = &stream{}, &stream{}

	var live []*segment

	for _, e := range entries {
		segID, compaction, ok := parseSegmentName(e.Name())
		if !ok {
			continue
		}

		sm, isListed := listed[segID]
		if !isListed && segID < saved.nextSeg {
			garbage = append(garbage, e.Name())

			continue
		}

		seg, err := openSegment(m.dir, segID, compaction)
		if err != nil {
			return nil, nil, nil, err
		}

		live = append(live, seg)
		vlog.nextSeg = max(vlog.nextSeg, segID+1)
		start := int64(0)

		if isListed {
			if seg.size < sm.size {
				return nil, nil, nil, fmt.Errorf("%w: segment %d is shorter than its checkpoint", ErrCorrupt, segID)
			}

			seg.dead, start = sm.dead, sm.size
			delete(listed, segID)
		}

		target := puts
		if compaction {
			target = moves
		}

		target.segs = append(target.segs, seg)
		target.starts = append(target.starts, start)
	}

	if len(listed) > 0 {
		return nil, nil, nil, fmt.Errorf("%w: %d segments of the checkpoint are missing", ErrCorrupt, len(listed))
	}

	slices.SortFunc(live, func(a, b *segment) int { return int(a.id) - int(b.id) })

	for _, seg := range live {
		vlog.total += seg.size
		vlog.dead += seg.dead
	}

	vlog.table.Store(newSegTable(live))

	for _, st := range []*stream{puts, moves} {
		if len(st.segs) > 0 {
			st.off = st.starts[0]
			st.advance(vlog)
		}
	}

	return puts, moves, garbage, nil
}

func parseSegmentName(name string) (uint32, bool, bool) {
	stem, compaction := strings.CutSuffix(name, compactionSuffix)
	if !compaction {
		var ok bool
		if stem, ok = strings.CutSuffix(name, segmentSuffix); !ok {
			return 0, false, false
		}
	}

	segID, err := strconv.ParseUint(stem, 10, 32)

	return uint32(segID), compaction, err == nil
}

// replay applies the records written after the checkpoint in the order they
// took effect. Puts and tombstones must follow one another without a gap; at
// the first missing one replay stops, so the map comes back as an exact
// prefix of what was written. A move applies only where the index still
// points at the record it copied, exactly as when it was first made.
func (m *Map) replay(saved *meta, puts, moves *stream) error {
	vlog := m.log
	expected := saved.seq
	lastMove := saved.moveSeq

	for {
		if puts.ok && recordSeq(puts.rec) != expected {
			puts.ok = false
		}

		if moves.ok && recordMove(moves.rec).at <= expected {
			err := m.replayMove(moves.rec, moves.addr())
			if err != nil {
				return err
			}

			lastMove = recordSeq(moves.rec) + 1
			moves.advance(vlog)

			continue
		}

		if !puts.ok {
			break
		}

		err := m.replayPut(puts.rec, puts.addr())
		if err != nil {
			return err
		}

		expected++
		puts.advance(vlog)
	}

	vlog.nextSeq, vlog.nextMoveSeq = expected, max(lastMove, saved.moveSeq)

	// Drop everything after the point where each stream stopped.
	for _, st := range []*stream{puts, moves} {
		err := m.truncateStream(st)
		if err != nil {
			return err
		}
	}

	return nil
}

// truncateStream cuts a stream's data after the point where replay stopped.
func (m *Map) truncateStream(st *stream) error {
	vlog := m.log

	for i := st.i; i < len(st.segs); i++ {
		seg := st.segs[i]

		end := st.starts[i]
		if i == st.i {
			end = st.off
		}

		if end >= seg.size {
			continue
		}

		vlog.total -= seg.size - end

		err := seg.truncate(end)
		if err != nil {
			return err
		}
	}

	return nil
}

// replayPut applies a put or tombstone found at addr.
func (m *Map) replayPut(rec []byte, addr loc) error {
	key := string(recordKey(rec))
	hash := m.hash(key, m.seed)
	shd := m.shardFor(hash)
	tombstone := recordKindOf(rec) == kindTombstone

	if tombstone {
		m.log.markDead(addr.seg, addr.size)
	}

	var checkBuf [2]keyCheck

	checks := checkBuf[:0]

	shd.mu.Lock()
	defer shd.mu.Unlock()

	for {
		ref, found, err := shd.find(m.log, hash, key, &checks)
		if errors.Is(err, errRetry) {
			continue
		}

		if err != nil {
			return err
		}

		return m.applyFound(shd, hash, ref, found, addr, tombstone)
	}
}

// applyFound makes the slot found for a key point at addr, or removes it for
// a tombstone.
func (m *Map) applyFound(shd *shard, hash uint64, ref slotRef, found bool, addr loc, tombstone bool) error {
	if found {
		m.log.markDead(ref.addr.seg, ref.addr.size)

		err := shd.pc.modify(ref.frame)
		if err != nil {
			return err
		}

		if tombstone {
			removeSlot(shd.pc.buf(ref.frame), ref.slot)
			shd.count--

			return nil
		}

		setSlotLoc(shd.pc.buf(ref.frame), ref.slot, addr)

		return nil
	}

	if tombstone {
		return nil
	}

	for {
		err := shd.ix.insert(hash, addr)
		if !errors.Is(err, errRetry) {
			if err == nil {
				shd.count++
			}

			return err
		}
	}
}

// replayMove applies a move found at addr if the index still points at the
// record it copied.
func (m *Map) replayMove(rec []byte, addr loc) error {
	info := recordMove(rec)
	hash := m.hash(string(recordKey(rec)), m.seed)
	shd := m.shardFor(hash)

	shd.mu.Lock()
	defer shd.mu.Unlock()

	ref, live, err := shd.liveSlot(hash, info.from.seg, int64(info.from.off))
	if err != nil {
		return err
	}

	if !live {
		m.log.markDead(addr.seg, addr.size)

		return nil
	}

	m.log.markDead(ref.addr.seg, ref.addr.size)

	err = shd.pc.modify(ref.frame)
	if err != nil {
		return err
	}

	setSlotLoc(shd.pc.buf(ref.frame), ref.slot, addr)

	return nil
}
