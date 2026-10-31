package diskmap

import (
	"errors"
	"fmt"
	"slices"
)

// A checkpoint is a consistent snapshot of the index, made durable. Taking
// one freezes writers only long enough to note sequence numbers, copy each
// shard's directory and page table, and mark the index pages whose newest
// content is not yet on disk. Those pages are then written to fresh slots in
// the background; a page that changes before its turn has its snapshot
// content queued first. Because index pages are copy-on-write, the previous
// checkpoint stays intact on disk until the new meta file replaces it.
//
// Recovery loads the last checkpoint and replays the records written after
// it, so the time it takes is bounded by the checkpoint interval, not by the
// size of the map.

// Sync makes every write that has returned so far durable: after Sync
// returns, those writes survive a power failure as well as a crash.
func (m *Map) Sync() error {
	if m.closed.Load() {
		return ErrClosed
	}

	return m.log.syncAll()
}

// checkpoint takes a checkpoint. If writeLocked is set the caller already
// holds writeMu for writing.
func (m *Map) checkpoint(writeLocked bool) error {
	m.checkpointMu.Lock()
	defer m.checkpointMu.Unlock()

	saved, gen, err := m.freeze(writeLocked)
	if err != nil {
		return err
	}

	err = m.writeSnapshot(saved)
	if err != nil {
		for _, shd := range m.shards {
			shd.mu.Lock()
			shd.pc.abortSnapshot()
			shd.mu.Unlock()
		}

		return err
	}

	for _, shd := range m.shards {
		shd.mu.Lock()
		shd.pc.commitSnapshot()
		shd.mu.Unlock()
	}

	m.dropRetired(gen)

	return nil
}

// freeze records the state the checkpoint captures and marks the pages it
// must write, with every writer stopped.
func (m *Map) freeze(writeLocked bool) (*meta, uint64, error) {
	vlog := m.log

	if !writeLocked {
		vlog.writeMu.Lock()
		defer vlog.writeMu.Unlock()
	}

	for _, shd := range m.shards {
		shd.mu.Lock()
		defer shd.mu.Unlock() //nolint:gocritic // Every shard stays locked while the state is copied.
	}

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	err := vlog.err()
	if err != nil {
		return nil, 0, err
	}

	saved := &meta{seed: m.seed, seq: vlog.nextSeq, moveSeq: vlog.nextMoveSeq, nextSeg: vlog.nextSeg}

	for _, seg := range vlog.table.Load().all() {
		saved.segments = append(saved.segments, segmentMeta{
			id: seg.id, compaction: seg.compaction, size: seg.size, dead: seg.dead,
		})
	}

	for _, shd := range m.shards {
		if shd.pc.failed != nil {
			return nil, 0, shd.pc.failed
		}

		saved.shards = append(saved.shards, shardMeta{
			count: uint64(shd.count), //nolint:gosec // Count is non-negative.
			depth: shd.ix.depth,
			dir:   slices.Clone(shd.ix.dir),
		})
		shd.pc.beginSnapshot()
	}

	// Writers move on to fresh segments, so syncing the frozen ones below
	// never makes a writer wait for an fsync.
	if vlog.active.size > 0 {
		err = vlog.rotate()
		if err != nil {
			return nil, 0, err
		}
	}

	vlog.compactOut = nil
	vlog.checkpointGen++
	vlog.sinceCheckpoint = 0

	return saved, vlog.checkpointGen, nil
}

// writeSnapshot writes the snapshot's index pages, makes the records it
// refers to durable, and commits the meta file.
func (m *Map) writeSnapshot(saved *meta) error {
	for i, shd := range m.shards {
		shd.mu.Lock()
		table, err := shd.pc.finishSnapshot()
		saved.shards[i].pageTable = slices.Clone(table)
		shd.mu.Unlock()

		if err != nil {
			return err
		}
	}

	for _, shd := range m.shards {
		err := shd.idxFile.Sync()
		if err != nil {
			return fmt.Errorf("diskmap: syncing index: %w", err)
		}
	}

	err := m.log.syncFrozen(saved)
	if err != nil {
		return err
	}

	err = writeMeta(m.dir, saved)
	if err != nil {
		return err
	}

	m.log.mu.Lock()
	m.log.checkpoints++
	m.log.mu.Unlock()

	return nil
}

// dropRetired deletes the files of segments retired before the checkpoint
// numbered gen froze the map; no durable checkpoint refers to them now.
func (m *Map) dropRetired(gen uint64) {
	vlog := m.log

	vlog.mu.Lock()

	var drop []*segment

	kept := vlog.retired[:0]

	for _, r := range vlog.retired {
		if r.gen < gen {
			drop = append(drop, r.seg)
		} else {
			kept = append(kept, r)
		}
	}

	vlog.retired = kept
	vlog.mu.Unlock()

	for _, seg := range drop {
		seg.drop()
	}
}

// checkpointUnlocked takes a checkpoint with the log lock released.
func (m *Map) checkpointUnlocked() error {
	m.log.mu.Unlock()
	defer m.log.mu.Lock()

	return m.checkpoint(false)
}

// checkpointLoop takes a checkpoint whenever enough has been written since
// the last one.
func (m *Map) checkpointLoop() {
	defer m.wg.Done()

	vlog := m.log

	vlog.mu.Lock()
	defer vlog.mu.Unlock()

	for {
		for !vlog.stopping && vlog.sinceCheckpoint < vlog.checkpointBytes {
			vlog.checkpointCond.Wait()
		}

		if vlog.stopping {
			return
		}

		err := m.checkpointUnlocked()
		if err != nil && !errors.Is(err, ErrClosed) {
			_ = vlog.fail(fmt.Errorf("diskmap: checkpoint: %w", err))
		}
	}
}
