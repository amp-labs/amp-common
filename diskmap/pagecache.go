package diskmap

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

const (
	noFrame = -1
	noPage  = math.MaxUint32

	flagRef   = 1 << 0
	flagDirty = 1 << 1

	minFrames = 8

	// One page in writeBackShare of each shard's budget, between minWriteBack
	// and maxWriteBack, buffers evicted dirty pages for the background writer.
	writeBackShare = 8
	minWriteBack   = 2
	maxWriteBack   = 32
)

// pagePool holds buffers for reading index pages with the lock released.
var pagePool = sync.Pool{New: func() any { return new([pageSize]byte) }}

// errRetry reports that a shard operation released the shard lock to do
// I/O. Anything the caller read under the lock before the call may be
// stale, so it must start over; the lock is held again when errRetry is
// returned, and the I/O's result is cached, so the retry makes progress.
var errRetry = errors.New("diskmap: retry")

// pageCache is a fixed-size cache of index pages with CLOCK eviction. All
// frames live in one pointer-free allocation, so the garbage collector never
// scans it and the cache never grows past its budget.
//
// No file I/O happens under the shard lock in the common case. Misses read
// the page with the lock released and return errRetry. Dirty pages are
// copied to a bounded write-back queue on eviction and written by a
// background goroutine; only when that queue is full does an eviction write
// synchronously.
type pageCache struct {
	mu   *sync.Mutex // the owning shard's lock
	file *os.File
	data []byte

	framePage []uint32 // page held by each frame, or noPage
	flags     []uint8  // flagRef and flagDirty per frame
	pins      []uint8  // frames that must not be evicted
	hand      int

	frameOf []int32  // frame holding each page, or noFrame
	version []uint32 // bumped each time a page is written to the file

	// epoch changes when the cache is reset or stopped, invalidating reads
	// that were in progress.
	epoch uint64

	// Write-back queue. wb holds the newest pending contents of each page
	// not yet in the file; readers must prefer it to the file.
	wb        map[uint32]*writeBack
	wbQueue   []uint32
	wbFree    [][]byte
	wbMax     int
	wbBusy    bool // the writer is writing a page with the lock released
	wbCond    *sync.Cond
	wbStopped bool

	hits   uint64
	misses uint64
	writes uint64

	// failed is set when a write-back fails; the index file may then be
	// missing changes, so the shard stops accepting operations.
	failed error
}

type writeBack struct {
	buf     []byte
	writing bool
}

// newPageCache creates a cache with a budget of the given number of pages,
// split between frames and write-back buffers.
func newPageCache(mu *sync.Mutex, file *os.File, budget int) *pageCache {
	wbPages := min(max(budget/writeBackShare, minWriteBack), maxWriteBack)
	frames := max(budget-wbPages, minFrames)

	cache := &pageCache{
		mu:        mu,
		file:      file,
		data:      make([]byte, frames*pageSize),
		framePage: make([]uint32, frames),
		flags:     make([]uint8, frames),
		pins:      make([]uint8, frames),
		wb:        make(map[uint32]*writeBack),
		wbMax:     wbPages,
		wbCond:    sync.NewCond(mu),
	}

	for i := range cache.framePage {
		cache.framePage[i] = noPage
	}

	wbData := make([]byte, wbPages*pageSize)
	for i := range wbPages {
		cache.wbFree = append(cache.wbFree, wbData[i*pageSize:(i+1)*pageSize:(i+1)*pageSize])
	}

	return cache
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
		pc.version = append(pc.version, 0)
	}
}

// page returns the frame holding a page. On a miss it reads the page with
// the lock released, caches it and returns errRetry.
func (pc *pageCache) page(pageID uint32) (int, error) {
	if frame := pc.frameOf[pageID]; frame != noFrame {
		pc.flags[frame] |= flagRef
		pc.hits++

		return int(frame), nil
	}

	pc.misses++

	if wb := pc.wb[pageID]; wb != nil {
		frame, err := pc.evict()
		if err != nil {
			return 0, err
		}

		// Clean, because wb.buf is on its way to the file.
		copy(pc.buf(frame), wb.buf)
		pc.install(frame, pageID, flagRef)

		return frame, nil
	}

	epoch, version := pc.epoch, pc.version[pageID]

	pooled, _ := pagePool.Get().(*[pageSize]byte)
	defer pagePool.Put(pooled)

	err := pc.readUnlocked(pooled[:], pageID)

	if pc.epoch != epoch {
		return 0, errRetry
	}

	if err != nil {
		return 0, fmt.Errorf("diskmap: reading index page %d: %w", pageID, err)
	}

	// Install only if nobody loaded, changed or rewrote the page meanwhile;
	// otherwise the copy may be stale and the retry will read it again.
	if pc.frameOf[pageID] == noFrame && pc.wb[pageID] == nil && pc.version[pageID] == version {
		frame, err := pc.evict()
		if err != nil {
			return 0, err
		}

		copy(pc.buf(frame), pooled[:])
		pc.install(frame, pageID, flagRef)
	}

	return 0, errRetry
}

func (pc *pageCache) readUnlocked(page []byte, pageID uint32) error {
	pc.mu.Unlock()
	defer pc.mu.Lock()

	_, err := pc.file.ReadAt(page, int64(pageID)*pageSize)

	return err
}

func (pc *pageCache) writeUnlocked(page []byte, pageID uint32) error {
	pc.mu.Unlock()
	defer pc.mu.Lock()

	_, err := pc.file.WriteAt(page, int64(pageID)*pageSize)

	return err
}

// newPage adds a page to the cache as an empty, dirty page without reading
// the file. The caller must have called setPages to cover it.
func (pc *pageCache) newPage(pageID uint32) (int, error) {
	frame, err := pc.evict()
	if err != nil {
		return 0, err
	}

	clear(pc.buf(frame))
	pc.install(frame, pageID, flagRef|flagDirty)

	return frame, nil
}

func (pc *pageCache) install(frame int, pageID uint32, flags uint8) {
	pc.framePage[frame] = pageID
	pc.flags[frame] = flags
	pc.frameOf[pageID] = int32(frame) //nolint:gosec // Frame counts fit in int32.
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

// evict picks a frame with the CLOCK algorithm, queues it for write-back if
// dirty and returns it empty.
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

		if pc.flags[frame]&flagDirty != 0 {
			err := pc.queueWriteBack(frame)
			if err != nil {
				return 0, err
			}
		}

		if pageID := pc.framePage[frame]; pageID != noPage {
			pc.frameOf[pageID] = noFrame
			pc.framePage[frame] = noPage
		}

		return frame, nil
	}
}

// queueWriteBack hands a dirty frame's contents to the background writer,
// or writes them synchronously if its queue is full.
func (pc *pageCache) queueWriteBack(frame int) error {
	pageID := pc.framePage[frame]
	pc.flags[frame] &^= flagDirty

	// A queued copy not yet being written can simply be updated.
	if wb := pc.wb[pageID]; wb != nil && !wb.writing {
		copy(wb.buf, pc.buf(frame))

		return nil
	}

	var buf []byte

	switch {
	case len(pc.wbFree) > 0:
		buf = pc.wbFree[len(pc.wbFree)-1]
		pc.wbFree = pc.wbFree[:len(pc.wbFree)-1]
	case pc.wb[pageID] != nil:
		// An older copy of this page is being written. Writing the newer one
		// synchronously could land first and be overwritten, so queue it in
		// a buffer beyond the usual bound; this is rare.
		buf = make([]byte, pageSize)
	default:
		return pc.writePage(pageID, pc.buf(frame))
	}

	copy(buf, pc.buf(frame))
	pc.wb[pageID] = &writeBack{buf: buf}
	pc.wbQueue = append(pc.wbQueue, pageID)
	pc.wbCond.Broadcast()

	return nil
}

// writePage writes a page to the file under the lock.
func (pc *pageCache) writePage(pageID uint32, page []byte) error {
	_, err := pc.file.WriteAt(page, int64(pageID)*pageSize)
	if err != nil {
		pc.failed = fmt.Errorf("diskmap: writing index page %d: %w", pageID, err)

		return pc.failed
	}

	pc.version[pageID]++
	pc.writes++

	return nil
}

// writeLoop is the background writer. It writes queued pages with the lock
// released.
func (pc *pageCache) writeLoop(wg *sync.WaitGroup) {
	defer wg.Done()

	pc.mu.Lock()
	defer pc.mu.Unlock()

	for {
		for len(pc.wbQueue) == 0 {
			if pc.wbStopped {
				return
			}

			pc.wbCond.Wait()
		}

		pageID := pc.wbQueue[0]
		pc.wbQueue = pc.wbQueue[1:]

		entry := pc.wb[pageID]
		if entry == nil {
			continue
		}

		entry.writing = true
		pc.wbBusy = true
		epoch := pc.epoch

		err := pc.writeUnlocked(entry.buf, pageID)

		pc.wbBusy = false

		if pc.epoch == epoch {
			if err != nil && pc.failed == nil {
				pc.failed = fmt.Errorf("diskmap: writing index page %d: %w", pageID, err)
			}

			pc.version[pageID]++
			pc.writes++

			// A newer copy may have been queued while this one was written.
			if pc.wb[pageID] == entry {
				delete(pc.wb, pageID)
			}
		}

		pc.recycle(entry.buf)
		pc.wbCond.Broadcast()
	}
}

// recycle returns a write-back buffer to the free list, dropping buffers
// allocated beyond the bound.
func (pc *pageCache) recycle(buf []byte) {
	if len(pc.wbFree) < pc.wbMax {
		pc.wbFree = append(pc.wbFree, buf)
	}
}

// idle waits until the background writer is not writing. The lock is
// released while waiting.
func (pc *pageCache) idle() {
	for pc.wbBusy {
		pc.wbCond.Wait()
	}
}

// flush writes every queued and dirty page to the file. The lock is released
// while waiting for the background writer.
func (pc *pageCache) flush() error {
	for (len(pc.wbQueue) > 0 || pc.wbBusy) && pc.failed == nil {
		pc.wbCond.Wait()
	}

	if pc.failed != nil {
		return pc.failed
	}

	for frame, pageID := range pc.framePage {
		if pc.flags[frame]&flagDirty == 0 {
			continue
		}

		err := pc.writePage(pageID, pc.buf(frame))
		if err != nil {
			return err
		}

		pc.flags[frame] &^= flagDirty
	}

	return nil
}

// stop tells the background writer to exit once its queue is empty.
func (pc *pageCache) stop() {
	pc.wbStopped = true
	pc.epoch++
	pc.wbCond.Broadcast()
}

// reset forgets every cached and queued page without writing anything. The
// background writer must be idle.
func (pc *pageCache) reset() {
	for i := range pc.framePage {
		pc.framePage[i] = noPage
		pc.flags[i] = 0
		pc.pins[i] = 0
	}

	for _, wb := range pc.wb {
		pc.recycle(wb.buf)
	}

	clear(pc.wb)
	pc.wbQueue = pc.wbQueue[:0]
	pc.frameOf = pc.frameOf[:0]
	pc.version = pc.version[:0]
	pc.hand = 0
	pc.epoch++
}
