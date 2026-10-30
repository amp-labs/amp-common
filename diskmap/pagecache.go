package diskmap

import (
	"fmt"
	"math"
	"os"
)

const (
	noFrame = -1
	noPage  = math.MaxUint32

	flagRef   = 1 << 0
	flagDirty = 1 << 1

	minFrames = 8
)

// pageCache is a fixed-size write-back cache of index pages with CLOCK
// eviction. All frames live in one pointer-free allocation, so the garbage
// collector never scans it and the cache never grows past its budget.
type pageCache struct {
	file *os.File
	data []byte

	framePage []uint32 // page held by each frame, or noPage
	flags     []uint8  // flagRef and flagDirty per frame
	pins      []uint8  // frames that must not be evicted
	frameOf   []int32  // frame holding each page, or noFrame
	hand      int

	hits   uint64
	misses uint64
	writes uint64

	// failed is set when a write-back fails; the index file may then be
	// missing changes, so the map stops accepting operations.
	failed error
}

func newPageCache(file *os.File, frames int) *pageCache {
	frames = max(frames, minFrames)

	pc := &pageCache{
		file:      file,
		data:      make([]byte, frames*pageSize),
		framePage: make([]uint32, frames),
		flags:     make([]uint8, frames),
		pins:      make([]uint8, frames),
	}

	for i := range pc.framePage {
		pc.framePage[i] = noPage
	}

	return pc
}

func (pc *pageCache) frames() int {
	return len(pc.framePage)
}

func (pc *pageCache) buf(frame int) []byte {
	return pc.data[frame*pageSize : (frame+1)*pageSize : (frame+1)*pageSize]
}

// setPages declares how many pages the index file holds.
func (pc *pageCache) setPages(n uint32) {
	for uint32(len(pc.frameOf)) < n { //nolint:gosec // Page counts fit in uint32.
		pc.frameOf = append(pc.frameOf, noFrame)
	}
}

// page returns the frame holding page id, reading it from disk on a miss.
func (pc *pageCache) page(id uint32) (int, error) {
	if frame := pc.frameOf[id]; frame != noFrame {
		pc.flags[frame] |= flagRef
		pc.hits++

		return int(frame), nil
	}

	pc.misses++

	frame, err := pc.evict()
	if err != nil {
		return 0, err
	}

	if _, err := pc.file.ReadAt(pc.buf(frame), int64(id)*pageSize); err != nil {
		return 0, fmt.Errorf("diskmap: reading index page %d: %w", id, err)
	}

	pc.install(frame, id, flagRef)

	return frame, nil
}

// newPage adds page id to the cache as an empty, dirty page without reading
// the file. The caller must have called setPages to cover id.
func (pc *pageCache) newPage(id uint32) (int, error) {
	frame, err := pc.evict()
	if err != nil {
		return 0, err
	}

	clear(pc.buf(frame))
	pc.install(frame, id, flagRef|flagDirty)

	return frame, nil
}

func (pc *pageCache) install(frame int, id uint32, flags uint8) {
	pc.framePage[frame] = id
	pc.flags[frame] = flags
	pc.frameOf[id] = int32(frame) //nolint:gosec // Frame counts fit in int32.
}

func (pc *pageCache) markDirty(frame int) {
	pc.flags[frame] |= flagDirty
}

func (pc *pageCache) pin(frame int) {
	pc.pins[frame]++
}

func (pc *pageCache) unpin(frame int) {
	pc.pins[frame]--
}

// evict picks a frame with the CLOCK algorithm, writes it back if dirty and
// returns it empty.
func (pc *pageCache) evict() (int, error) {
	for {
		frame := pc.hand

		pc.hand++
		if pc.hand == len(pc.framePage) {
			pc.hand = 0
		}

		if pc.pins[frame] > 0 {
			continue
		}

		if pc.flags[frame]&flagRef != 0 {
			pc.flags[frame] &^= flagRef

			continue
		}

		if err := pc.writeBack(frame); err != nil {
			return 0, err
		}

		if id := pc.framePage[frame]; id != noPage {
			pc.frameOf[id] = noFrame
			pc.framePage[frame] = noPage
		}

		return frame, nil
	}
}

func (pc *pageCache) writeBack(frame int) error {
	if pc.flags[frame]&flagDirty == 0 {
		return nil
	}

	id := pc.framePage[frame]
	if _, err := pc.file.WriteAt(pc.buf(frame), int64(id)*pageSize); err != nil {
		pc.failed = fmt.Errorf("diskmap: writing index page %d: %w", id, err)

		return pc.failed
	}

	pc.flags[frame] &^= flagDirty
	pc.writes++

	return nil
}

// flush writes every dirty page back to the file.
func (pc *pageCache) flush() error {
	for frame := range pc.framePage {
		if err := pc.writeBack(frame); err != nil {
			return err
		}
	}

	return nil
}

// reset forgets every cached page without writing anything.
func (pc *pageCache) reset() {
	for i := range pc.framePage {
		pc.framePage[i] = noPage
		pc.flags[i] = 0
		pc.pins[i] = 0
	}

	pc.frameOf = pc.frameOf[:0]
	pc.hand = 0
}
