package diskmap

import (
	"fmt"
)

// maxDepth caps the directory at 2^maxDepth entries. With slotsPerPage slots
// per page a well-mixed hash reaches it only past hundreds of billions of
// keys, so hitting it means the hash has stopped spreading keys out.
const maxDepth = 36

// index maps 64-bit key hashes to record locations using extendible hashing.
// The directory, indexed by the top depth bits of a hash, lives in memory and
// costs four bytes per entry. Pages live in the index file behind a pageCache.
// A lookup always touches exactly one page, and growth splits one page at a
// time, so no operation ever rehashes the whole table.
type index struct {
	pc       *pageCache
	dir      []uint32
	depth    uint8
	numPages uint32
}

// newIndex creates an empty index with a single page.
func newIndex(pc *pageCache) (*index, error) {
	ix := &index{pc: pc}

	return ix, ix.reset()
}

func (ix *index) reset() error {
	ix.pc.reset()
	ix.dir = []uint32{0}
	ix.depth = 0
	ix.numPages = 1
	ix.pc.setPages(1)

	_, err := ix.pc.newPage(0)

	return err
}

// pageFor returns the page responsible for hash h.
func (ix *index) pageFor(h uint64) uint32 {
	return ix.dir[h>>(64-ix.depth)]
}

// frameFor returns the cache frame holding the page responsible for hash h.
// The frame stays valid until the next index or page cache call.
func (ix *index) frameFor(h uint64) (int, error) {
	return ix.pc.page(ix.pageFor(h))
}

// insert adds a slot for hash h, splitting pages as needed.
func (ix *index) insert(h uint64, l loc) error {
	for {
		id := ix.pageFor(h)

		frame, err := ix.pc.page(id)
		if err != nil {
			return err
		}

		p := ix.pc.buf(frame)

		n := pageCount(p)
		if n < slotsPerPage {
			setSlot(p, n, h, l)
			setPageCount(p, n+1)
			ix.pc.markDirty(frame)

			return nil
		}

		if err := ix.split(id, frame, h); err != nil {
			return err
		}
	}
}

// split divides a full page in two on the next hash bit below its local
// depth, doubling the directory first if the page already uses every
// directory bit. h is any hash that maps to the page.
func (ix *index) split(id uint32, frame int, h uint64) error {
	p := ix.pc.buf(frame)
	depth := pageDepth(p)

	if allSameHash(p) {
		return fmt.Errorf("%w: %d keys share one 64-bit hash", ErrTooManyCollisions, pageCount(p))
	}

	if depth == ix.depth {
		if ix.depth == maxDepth {
			return fmt.Errorf("%w: index directory reached 2^%d entries", ErrTooManyCollisions, maxDepth)
		}

		ix.grow()
	}

	ix.pc.pin(frame)
	defer ix.pc.unpin(frame)

	newID := ix.numPages
	ix.numPages++
	ix.pc.setPages(ix.numPages)

	newFrame, err := ix.pc.newPage(newID)
	if err != nil {
		return err
	}

	q := ix.pc.buf(newFrame)
	bit := uint64(1) << (63 - depth)
	kept, moved := 0, 0

	for i := range pageCount(p) {
		sh, sl := slotHash(p, i), slotLoc(p, i)
		if sh&bit != 0 {
			setSlot(q, moved, sh, sl)
			moved++
		} else {
			setSlot(p, kept, sh, sl)
			kept++
		}
	}

	setPageCount(p, kept)
	setPageCount(q, moved)
	setPageDepth(p, depth+1)
	setPageDepth(q, depth+1)
	ix.pc.markDirty(frame)

	// The page owns a contiguous run of 2^(global-local) directory entries;
	// the upper half of that run now belongs to the new page.
	shift := ix.depth - depth
	span := uint64(1) << shift
	start := (h >> (64 - ix.depth)) &^ (span - 1)

	for i := start + span/2; i < start+span; i++ {
		ix.dir[i] = newID
	}

	return nil
}

// grow doubles the directory. Each entry is duplicated, so every page keeps
// owning the same hashes.
func (ix *index) grow() {
	dir := make([]uint32, 2*len(ix.dir))
	for i, id := range ix.dir {
		dir[2*i] = id
		dir[2*i+1] = id
	}

	ix.dir = dir
	ix.depth++
}

func allSameHash(p []byte) bool {
	n := pageCount(p)
	if n == 0 {
		return false
	}

	first := slotHash(p, 0)
	for i := 1; i < n; i++ {
		if slotHash(p, i) != first {
			return false
		}
	}

	return true
}
