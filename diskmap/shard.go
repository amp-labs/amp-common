package diskmap

import (
	"errors"
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
	vc      *valueCache // nil unless WithValueCache is used
	count   int
	closed  bool
}

// slotRef identifies a slot in a cached index page. It is valid until the
// next index call or until the lock is released.
type slotRef struct {
	frame int
	slot  int
	addr  loc
}

// keyCheck remembers whether the record at a location holds the key being
// looked up, so a retried find does not read it again.
type keyCheck struct {
	addr loc
	same bool
}

func indexPath(dir string, id int) string {
	return filepath.Join(dir, fmt.Sprintf("%03d%s", id, indexSuffix))
}

func openShard(dir string, id, pages, cacheBytes int) (*shard, error) {
	file, err := os.OpenFile(indexPath(dir, id), os.O_RDWR|os.O_CREATE, filePerm) //nolint:gosec // Path built from dir.
	if err != nil {
		return nil, fmt.Errorf("diskmap: opening index: %w", err)
	}

	sh := &shard{id: id, idxFile: file}
	sh.pc = newPageCache(&sh.mu, file, pages)

	if cacheBytes > 0 {
		sh.vc = newValueCache(cacheBytes)
	}

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

func (sh *shard) usable(vlog *valueLog) error {
	if sh.closed {
		return ErrClosed
	}

	err := vlog.err()
	if err != nil {
		return err
	}

	return sh.pc.failed
}

// candidates returns the location of the one slot whose hash is h, or the
// number of such slots if it is not exactly one.
func (sh *shard) candidates(h uint64) (addr loc, n int, err error) {
	frame, err := sh.ix.frameFor(h)
	if err != nil {
		return loc{}, 0, err
	}

	page := sh.pc.buf(frame)
	mt := matchSlots(page, h)

	for i := mt.next(); i >= 0; i = mt.next() {
		addr = slotLoc(page, i)
		n++
	}

	return addr, n, nil
}

// find locates the slot for key. Slots whose hash matches are confirmed by
// reading the key stored in their record; reading one from a file releases
// the lock and returns errRetry, with the answer saved in checks.
func (sh *shard) find(vlog *valueLog, h uint64, key string, checks *[]keyCheck) (slotRef, bool, error) {
	frame, err := sh.ix.frameFor(h)
	if err != nil {
		return slotRef{}, false, err
	}

	page := sh.pc.buf(frame)
	mt := matchSlots(page, h)

	for slot := mt.next(); slot >= 0; slot = mt.next() {
		addr := slotLoc(page, slot)

		same, checked := checkedBefore(*checks, addr)
		if !checked {
			same, err = sh.checkKey(vlog, addr, key)
			*checks = append(*checks, keyCheck{addr: addr, same: same})

			if err != nil {
				return slotRef{}, false, err
			}
		}

		if same {
			return slotRef{frame: frame, slot: slot, addr: addr}, true, nil
		}
	}

	return slotRef{frame: frame}, false, nil
}

func checkedBefore(checks []keyCheck, addr loc) (same, checked bool) {
	for _, c := range checks {
		if c.addr == addr {
			return c.same, true
		}
	}

	return false, false
}

// checkKey reports whether the record at addr stores key, reading only its
// header and key. Records still in a write buffer are checked under the
// lock; reading one from a file releases the lock and returns errRetry.
func (sh *shard) checkKey(vlog *valueLog, addr loc, key string) (bool, error) {
	n := recordSize(len(key), 0)
	if int(addr.size) < n {
		return false, nil
	}

	seg := vlog.segment(addr.seg)
	if seg == nil {
		return false, fmt.Errorf("%w: index points to missing segment %d", ErrCorrupt, addr.seg)
	}

	buf := make([]byte, n)

	var err error

	if int64(addr.off)+int64(n) > seg.written.Load() {
		err = vlog.read(seg, int64(addr.off), buf)
	} else {
		err = sh.readUnlocked(seg, int64(addr.off), buf)
		if err == nil {
			err = errRetry
		}
	}

	if err != nil && !errors.Is(err, errRetry) {
		return false, err
	}

	keyLen, _ := recordLengths(buf)

	return keyLen == len(key) && string(buf[recordHeaderSize:]) == key, err
}

// readUnlocked reads from a segment file with the shard lock released. The
// index points into seg, so it cannot be dropped while the lock is held; the
// reference keeps it open once the lock is released.
func (sh *shard) readUnlocked(seg *segment, off int64, buf []byte) error {
	seg.acquire()
	defer seg.release()

	sh.mu.Unlock()
	defer sh.mu.Lock()

	return readFile(seg, off, buf)
}

// liveSlot finds the slot pointing at (seg, off), retrying after any I/O. A
// record is live exactly when such a slot exists. Callers must not depend on
// state they read before calling it.
func (sh *shard) liveSlot(h uint64, seg uint32, off int64) (slotRef, bool, error) {
	for {
		ref, live, err := sh.liveSlotOnce(h, seg, off)
		if !errors.Is(err, errRetry) {
			return ref, live, err
		}
	}
}

func (sh *shard) liveSlotOnce(h uint64, seg uint32, off int64) (slotRef, bool, error) {
	frame, err := sh.ix.frameFor(h)
	if err != nil {
		return slotRef{}, false, err
	}

	page := sh.pc.buf(frame)
	mt := matchSlots(page, h)

	for slot := mt.next(); slot >= 0; slot = mt.next() {
		if addr := slotLoc(page, slot); addr.seg == seg && int64(addr.off) == off {
			return slotRef{frame: frame, slot: slot, addr: addr}, true, nil
		}
	}

	return slotRef{}, false, nil
}

// reset empties the index. The page cache's writer must be idle.
func (sh *shard) reset() error {
	err := sh.idxFile.Truncate(0)
	if err != nil {
		return fmt.Errorf("diskmap: truncating index: %w", err)
	}

	sh.count = 0

	if sh.vc != nil {
		sh.vc.reset()
	}

	return sh.ix.reset()
}

// persist writes back every page and syncs the index file. It releases the
// lock while waiting for the background writer, so the shard must already
// be closed to other operations.
func (sh *shard) persist() (shardMeta, error) {
	err := sh.pc.flush()
	if err != nil {
		return shardMeta{}, err
	}

	err = sh.idxFile.Sync()
	if err != nil {
		return shardMeta{}, fmt.Errorf("diskmap: syncing index: %w", err)
	}

	return shardMeta{
		count:    uint64(sh.count), //nolint:gosec // Count is non-negative.
		numPages: sh.ix.numPages,
		depth:    sh.ix.depth,
		dir:      sh.ix.dir,
	}, nil
}
