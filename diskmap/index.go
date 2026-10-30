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
	return ix.dir[h>>(hashBits-ix.depth)]
}

// frameFor returns the cache frame holding the page responsible for hash h.
// The frame stays valid until the next index or page cache call. It returns
// errRetry if it had to read the page with the lock released.
func (ix *index) frameFor(h uint64) (int, error) {
	return ix.pc.page(ix.pageFor(h))
}

// insert adds a slot for a hash, splitting pages as needed.
func (ix *index) insert(hash uint64, addr loc) error {
	for {
		frame, err := ix.pc.page(ix.pageFor(hash))
		if err != nil {
			return err
		}

		page := ix.pc.buf(frame)

		n := pageCount(page)
		if n < slotsPerPage {
			setSlot(page, n, hash, addr)
			setPageCount(page, n+1)
			ix.pc.markDirty(frame)

			return nil
		}

		err = ix.split(frame, hash)
		if err != nil {
			return err
		}
	}
}

// split divides a full page in two on the next hash bit below its local
// depth, doubling the directory first if the page already uses every
// directory bit. hash is any hash that maps to the page.
func (ix *index) split(frame int, hash uint64) error {
	page := ix.pc.buf(frame)
	depth := pageDepth(page)

	if allSameHash(page) {
		return fmt.Errorf("%w: %d keys share one 64-bit hash", ErrTooManyCollisions, pageCount(page))
	}

	if depth == ix.depth {
		if ix.depth == maxDepth {
			return fmt.Errorf("%w: index directory reached 2^%d entries", ErrTooManyCollisions, maxDepth)
		}

		ix.grow()
	}

	ix.pc.pin(frame)
	defer ix.pc.unpin(frame)

	siblingID := ix.numPages
	ix.numPages++
	ix.pc.setPages(ix.numPages)

	siblingFrame, err := ix.pc.newPage(siblingID)
	if err != nil {
		return err
	}

	sibling := ix.pc.buf(siblingFrame)
	bit := uint64(1) << (hashBits - 1 - depth)
	kept, moved := 0, 0

	for i := range pageCount(page) {
		slotH, slotL := slotHash(page, i), slotLoc(page, i)
		if slotH&bit != 0 {
			setSlot(sibling, moved, slotH, slotL)
			moved++
		} else {
			setSlot(page, kept, slotH, slotL)
			kept++
		}
	}

	setPageCount(page, kept)
	setPageCount(sibling, moved)
	setPageDepth(page, depth+1)
	setPageDepth(sibling, depth+1)
	ix.pc.markDirty(frame)

	// The page owns a contiguous run of 2^(global-local) directory entries;
	// the upper half of that run now belongs to the sibling.
	span := uint64(1) << (ix.depth - depth)
	start := (hash >> (hashBits - ix.depth)) &^ (span - 1)

	for i := start + span>>1; i < start+span; i++ {
		ix.dir[i] = siblingID
	}

	return nil
}

// grow doubles the directory. Each entry is duplicated, so every page keeps
// owning the same hashes.
func (ix *index) grow() {
	dir := make([]uint32, len(ix.dir)<<1)
	for i, id := range ix.dir {
		dir[i<<1] = id
		dir[i<<1|1] = id
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
