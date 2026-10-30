package diskmap

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const indexSuffix = ".index"

// shard owns one slice of the key space: its own index file, page cache and
// lock. Keys are assigned to shards by the low bits of their hash, so
// operations on different shards never contend.
//
// Shard methods are called with the lock held. Those that may need file I/O
// release the lock for it and then return errRetry; callers loop until they
// get another result, re-checking anything they read before the call.
type shard struct {
	mu sync.Mutex

	id      int
	idxFile *os.File
	pc      *pageCache
	ix      *index
	count   int
	closed  bool
}

func indexPath(dir string, id int) string {
	return filepath.Join(dir, fmt.Sprintf("%03d%s", id, indexSuffix))
}

func openShard(dir string, id, pages int) (*shard, error) {
	file, err := os.OpenFile(indexPath(dir, id), os.O_RDWR|os.O_CREATE, filePerm) //nolint:gosec // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: opening index: %w", err)
	}

	sh := &shard{id: id, idxFile: file}
	sh.pc = newPageCache(&sh.mu, file, pages)

	return sh, nil
}

// create starts an empty index.
func (sh *shard) create() error {
	ix, err := newIndex(sh.pc)
	if err != nil {
		return err
	}

	sh.ix = ix

	return nil
}

// load restores the index from saved state.
func (sh *shard) load(saved shardMeta) error {
	info, err := sh.idxFile.Stat()
	if err != nil {
		return fmt.Errorf("diskmap: opening index: %w", err)
	}

	if info.Size() < int64(saved.numPages)*pageSize {
		return fmt.Errorf("%w: index file %d is shorter than its %d pages", ErrCorrupt, sh.id, saved.numPages)
	}

	sh.ix = &index{pc: sh.pc, dir: saved.dir, depth: saved.depth, numPages: saved.numPages}
	sh.pc.setPages(saved.numPages)
	sh.count = int(saved.count) //nolint:gosec // Written from an int.

	return nil
}

func (sh *shard) usable(lg *valueLog) error {
	if sh.closed {
		return ErrClosed
	}

	if err := lg.err(); err != nil {
		return err
	}

	return sh.pc.failed
}

// candidates returns the location of the one slot whose hash is h, or the
// number of such slots if it is not exactly one.
func (sh *shard) candidates(h uint64) (l loc, n int, err error) {
	frame, err := sh.ix.frameFor(h)
	if err != nil {
		return loc{}, 0, err
	}

	p := sh.pc.buf(frame)
	mt := matchSlots(p, h)

	for i := mt.next(); i >= 0; i = mt.next() {
		l = slotLoc(p, i)
		n++
	}

	return l, n, nil
}

// keyCheck remembers whether the record at a location holds the key being
// looked up, so a retried find does not read it again.
type keyCheck struct {
	l    loc
	same bool
}

// find locates the slot for key. Slots whose hash matches are confirmed by
// reading the key stored in their record; reading one from a file releases
// the lock and returns errRetry, with the answer saved in checks.
func (sh *shard) find(lg *valueLog, h uint64, key string, checks *[]keyCheck) (frame, slot int, l loc, found bool, err error) {
	frame, err = sh.ix.frameFor(h)
	if err != nil {
		return 0, 0, loc{}, false, err
	}

	p := sh.pc.buf(frame)
	mt := matchSlots(p, h)

next:
	for i := mt.next(); i >= 0; i = mt.next() {
		l = slotLoc(p, i)

		for _, c := range *checks {
			if c.l == l {
				if c.same {
					return frame, i, l, true, nil
				}

				continue next
			}
		}

		same, err := sh.checkKey(lg, l, key)
		*checks = append(*checks, keyCheck{l: l, same: same})

		if err != nil {
			return 0, 0, loc{}, false, err
		}

		if same {
			return frame, i, l, true, nil
		}
	}

	return frame, 0, loc{}, false, nil
}

// checkKey reports whether the record at l stores key, reading only its
// header and key. Records still in a write buffer are checked under the
// lock; reading one from a file releases the lock and returns errRetry.
func (sh *shard) checkKey(lg *valueLog, l loc, key string) (bool, error) {
	n := recordSize(len(key), 0)
	if int(l.size) < n {
		return false, nil
	}

	s := lg.segment(l.seg)
	if s == nil {
		return false, fmt.Errorf("%w: index points to missing segment %d", ErrCorrupt, l.seg)
	}

	buf := make([]byte, n)

	var err error

	if int64(l.off)+int64(n) > s.written.Load() {
		err = lg.read(s, int64(l.off), buf)
	} else {
		// The index points into s, so it cannot be dropped while the lock
		// is held; the reference keeps it open after the lock is released.
		s.acquire()
		sh.mu.Unlock()
		err = readFile(s, int64(l.off), buf)
		sh.mu.Lock()
		s.release()

		if err == nil {
			err = errRetry
		}
	}

	if err != nil && err != errRetry { //nolint:errorlint // errRetry is never wrapped.
		return false, err
	}

	keyLen, _ := recordLengths(buf)

	return keyLen == len(key) && string(buf[recordHeaderSize:]) == key, err
}

// liveSlot finds the slot pointing at (seg, off), if any. A record is live
// exactly when such a slot exists.
func (sh *shard) liveSlot(h uint64, seg uint32, off int64) (frame, slot int, live bool, err error) {
	frame, err = sh.ix.frameFor(h)
	if err != nil {
		return 0, 0, false, err
	}

	p := sh.pc.buf(frame)
	mt := matchSlots(p, h)

	for i := mt.next(); i >= 0; i = mt.next() {
		if l := slotLoc(p, i); l.seg == seg && int64(l.off) == off {
			return frame, i, true, nil
		}
	}

	return frame, 0, false, nil
}

// liveSlotRetry is liveSlot, retrying after any I/O. Callers must not depend
// on state they read before calling it.
func (sh *shard) liveSlotRetry(h uint64, seg uint32, off int64) (frame, slot int, live bool, err error) {
	for {
		frame, slot, live, err = sh.liveSlot(h, seg, off)
		if err != errRetry { //nolint:errorlint // errRetry is never wrapped.
			return frame, slot, live, err
		}
	}
}

// reset empties the index. The page cache's writer must be idle.
func (sh *shard) reset() error {
	if err := sh.idxFile.Truncate(0); err != nil {
		return fmt.Errorf("diskmap: truncating index: %w", err)
	}

	sh.count = 0

	return sh.ix.reset()
}

// persist writes back every page and syncs the index file. It releases the
// lock while waiting for the background writer, so the shard must already
// be closed to other operations.
func (sh *shard) persist() (shardMeta, error) {
	if err := sh.pc.flush(); err != nil {
		return shardMeta{}, err
	}

	if err := sh.idxFile.Sync(); err != nil {
		return shardMeta{}, fmt.Errorf("diskmap: syncing index: %w", err)
	}

	return shardMeta{
		count:    uint64(sh.count), //nolint:gosec // Count is non-negative.
		numPages: sh.ix.numPages,
		depth:    sh.ix.depth,
		dir:      sh.ix.dir,
	}, nil
}
