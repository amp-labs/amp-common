package diskmap

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"os"
	"sync"
)

const (
	noFrame = -1
	noPage  = math.MaxUint32
	noPhys  = math.MaxUint32

	flagRef   = 1 << 0
	flagDirty = 1 << 1
	flagSnap  = 1 << 2 // content belongs to the checkpoint being written

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
// Pages are copy-on-write on disk. The index refers to logical pages, and a
// page table maps each to a physical slot in the file. Slots that belong to
// a checkpoint are never overwritten; a page whose slot is protected moves to
// a fresh slot the next time it is written. So the file always holds the
// last durable checkpoint intact, whatever happened since.
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

	framePage []uint32 // logical page held by each frame, or noPage
	flags     []uint8  // flagRef, flagDirty and flagSnap per frame
	pins      []uint8  // frames that must not be evicted
	hand      int

	frameOf   []int32  // frame holding each logical page, or noFrame
	version   []uint32 // bumped each time a page is written to the file
	pageTable []uint32 // physical slot of each logical page, or noPhys

	physSlots uint32   // slots in the file
	freeSlots []uint32 // slots no checkpoint or page uses
	protected bitset   // slots of the durable checkpoint and the one in progress

	// The checkpoint being written: the slot of every page as of the
	// snapshot, the slots it newly protects, and pages still to be written.
	snapTable   []uint32
	snapSlots   bitset
	snapPending int
	snapCond    *sync.Cond

	// epoch changes when the cache is reset or stopped, invalidating reads
	// that were in progress.
	epoch uint64

	// Write-back queue. wb holds the newest pending contents of each page
	// not yet in the file; readers must prefer it to the file.
	wb        map[uint32]*writeBack // newest pending content of each page
	wbQueue   []*writeBack          // pending writes, oldest first
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
	pageID  uint32
	buf     []byte
	writing bool
	snap    bool // this content belongs to the checkpoint being written
}

// bitset is a growable set of slot numbers.
type bitset []uint64

const wordBits = 64

func (b *bitset) set(i uint32) {
	for int(i/wordBits) >= len(*b) {
		*b = append(*b, 0)
	}

	(*b)[i/wordBits] |= 1 << (i % wordBits)
}

func (b *bitset) has(i uint32) bool {
	return int(i/wordBits) < len(*b) && (*b)[i/wordBits]&(1<<(i%wordBits)) != 0
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
		snapCond:  sync.NewCond(mu),
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

// setPages declares how many logical pages the index has.
func (pc *pageCache) setPages(n uint32) {
	for uint32(len(pc.frameOf)) < n { //nolint:gosec // Page counts fit in uint32.
		pc.frameOf = append(pc.frameOf, noFrame)
		pc.version = append(pc.version, 0)
		pc.pageTable = append(pc.pageTable, noPhys)
	}
}

// load restores the page table of a checkpoint. Every slot it uses is
// protected; every other slot in the file is free.
func (pc *pageCache) load(table []uint32, physSlots uint32) {
	pc.setPages(uint32(len(table))) //nolint:gosec // Page counts fit in uint32.
	copy(pc.pageTable, table)
	pc.physSlots = physSlots
	pc.protected = nil

	for _, slot := range table {
		pc.protected.set(slot)
	}

	pc.freeSlots = pc.freeSlots[:0]

	for slot := range physSlots {
		if !pc.protected.has(slot) {
			pc.freeSlots = append(pc.freeSlots, slot)
		}
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

	slot := pc.pageTable[pageID]
	if slot == noPhys {
		return 0, fmt.Errorf("%w: index page %d was never written", ErrCorrupt, pageID)
	}

	epoch, version := pc.epoch, pc.version[pageID]

	pooled, _ := pagePool.Get().(*[pageSize]byte)
	defer pagePool.Put(pooled)

	err := pc.readUnlocked(pooled[:], slot)

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

func (pc *pageCache) readUnlocked(page []byte, slot uint32) error {
	pc.mu.Unlock()
	defer pc.mu.Lock()

	_, err := pc.file.ReadAt(page, int64(slot)*pageSize)

	return err
}

func (pc *pageCache) writeUnlocked(page []byte, slot uint32) error {
	pc.mu.Unlock()
	defer pc.mu.Lock()

	_, err := pc.file.WriteAt(page, int64(slot)*pageSize)

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

// modify must be called before changing a frame's page. If the page's
// current content belongs to a checkpoint being written, that content is
// queued for the checkpoint first.
func (pc *pageCache) modify(frame int) error {
	if pc.flags[frame]&flagSnap != 0 {
		err := pc.queueWriteBack(frame)
		if err != nil {
			return err
		}
	}

	pc.flags[frame] |= flagDirty

	return nil
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

		if pc.flags[frame]&(flagDirty|flagSnap) != 0 {
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

// queueWriteBack hands a frame's contents to the background writer, or
// writes them synchronously if its queue is full. The frame is clean
// afterwards.
func (pc *pageCache) queueWriteBack(frame int) error {
	pageID := pc.framePage[frame]
	snap := pc.flags[frame]&flagSnap != 0
	pc.flags[frame] &^= flagDirty | flagSnap

	// A queued copy not yet being written can simply be updated, unless it
	// already holds checkpoint content that this newer content must not
	// replace.
	if wb := pc.wb[pageID]; wb != nil && !wb.writing && (!wb.snap || snap) {
		copy(wb.buf, pc.buf(frame))
		wb.snap = wb.snap || snap

		return nil
	}

	var buf []byte

	switch {
	case len(pc.wbFree) > 0:
		buf = pc.wbFree[len(pc.wbFree)-1]
		pc.wbFree = pc.wbFree[:len(pc.wbFree)-1]
	case pc.wb[pageID] != nil:
		// An older copy of this page is queued or being written. Writing the
		// newer one synchronously could land first and be overwritten, so
		// queue it in a buffer beyond the usual bound; this is rare.
		buf = make([]byte, pageSize)
	default:
		return pc.writePage(pageID, pc.buf(frame), snap)
	}

	copy(buf, pc.buf(frame))

	entry := &writeBack{pageID: pageID, buf: buf, snap: snap}
	pc.wb[pageID] = entry
	pc.wbQueue = append(pc.wbQueue, entry)
	pc.wbCond.Broadcast()

	return nil
}

// slotFor returns the slot a page's next write goes to, moving the page to a
// fresh slot if its current one is protected or unassigned.
func (pc *pageCache) slotFor(pageID uint32) uint32 {
	slot := pc.pageTable[pageID]
	if slot != noPhys && !pc.protected.has(slot) && !pc.snapSlots.has(slot) {
		return slot
	}

	if n := len(pc.freeSlots); n > 0 {
		slot = pc.freeSlots[n-1]
		pc.freeSlots = pc.freeSlots[:n-1]
	} else {
		slot = pc.physSlots
		pc.physSlots++
	}

	pc.pageTable[pageID] = slot

	return slot
}

// written records that a page's content reached slot; snap marks content
// that belongs to the checkpoint being written.
func (pc *pageCache) written(pageID, slot uint32, snap bool) {
	pc.version[pageID]++
	pc.writes++

	if snap && pc.snapTable != nil {
		pc.snapTable[pageID] = slot
		pc.snapSlots.set(slot)
		pc.snapPending--

		if pc.snapPending == 0 {
			pc.snapCond.Broadcast()
		}
	}
}

// writePage writes a page to the file under the lock.
func (pc *pageCache) writePage(pageID uint32, page []byte, snap bool) error {
	slot := pc.slotFor(pageID)

	_, err := pc.file.WriteAt(page, int64(slot)*pageSize)
	if err != nil {
		pc.failed = fmt.Errorf("diskmap: writing index page %d: %w", pageID, err)
		pc.snapCond.Broadcast()

		return pc.failed
	}

	pc.written(pageID, slot, snap)

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

		entry := pc.wbQueue[0]
		pc.wbQueue = pc.wbQueue[1:]
		pageID := entry.pageID

		entry.writing = true
		pc.wbBusy = true
		epoch := pc.epoch
		slot := pc.slotFor(pageID)

		err := pc.writeUnlocked(entry.buf, slot)

		pc.wbBusy = false

		if pc.epoch == epoch {
			if err != nil && pc.failed == nil {
				pc.failed = fmt.Errorf("diskmap: writing index page %d: %w", pageID, err)
				pc.snapCond.Broadcast()
			}

			pc.written(pageID, slot, entry.snap)

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

// beginSnapshot starts a checkpoint of the index as it is now. Pages whose
// newest content is already on disk keep their slots, which become
// protected; every other page is marked, and its content is written to a
// fresh slot before it may change. The caller must hold the lock, and no
// other checkpoint may be in progress.
func (pc *pageCache) beginSnapshot() {
	pc.snapTable = make([]uint32, len(pc.pageTable))
	copy(pc.snapTable, pc.pageTable)
	pc.snapSlots = nil
	pc.snapPending = 0

	for pageID := range pc.pageTable {
		logical := uint32(pageID) //nolint:gosec // Page counts fit in uint32.

		if frame := pc.frameOf[logical]; frame != noFrame && pc.flags[frame]&flagDirty != 0 {
			pc.flags[frame] |= flagSnap
			pc.snapPending++

			continue
		}

		if wb := pc.wb[logical]; wb != nil {
			wb.snap = true
			pc.snapPending++

			continue
		}

		pc.snapSlots.set(pc.snapTable[logical])
	}
}

// finishSnapshot writes every page the checkpoint still needs and returns
// its page table. The lock is released while waiting for the writer.
func (pc *pageCache) finishSnapshot() ([]uint32, error) {
	for frame := range pc.framePage {
		if pc.flags[frame]&flagSnap != 0 {
			err := pc.queueWriteBack(frame)
			if err != nil {
				return nil, err
			}
		}
	}

	for pc.snapPending > 0 && pc.failed == nil {
		pc.snapCond.Wait()
	}

	if pc.failed != nil {
		return nil, pc.failed
	}

	return pc.snapTable, nil
}

// commitSnapshot makes the finished checkpoint the protected one: slots that
// only the previous checkpoint used, and no live page uses, become free.
func (pc *pageCache) commitSnapshot() {
	var live bitset

	for _, slot := range pc.pageTable {
		if slot != noPhys {
			live.set(slot)
		}
	}

	old := pc.protected
	pc.protected = pc.snapSlots

	for word, gone := range old {
		if word < len(pc.protected) {
			gone &^= pc.protected[word]
		}

		if word < len(live) {
			gone &^= live[word]
		}

		for gone != 0 {
			bit := bits.TrailingZeros64(gone)
			gone &= gone - 1

			pc.freeSlots = append(pc.freeSlots, uint32(word*wordBits+bit)) //nolint:gosec // Slot counts fit in uint32.
		}
	}

	pc.snapTable = nil
	pc.snapSlots = nil
}

// abortSnapshot abandons a checkpoint that failed.
func (pc *pageCache) abortSnapshot() {
	for frame := range pc.framePage {
		if pc.flags[frame]&flagSnap != 0 {
			pc.flags[frame] = pc.flags[frame]&^flagSnap | flagDirty
		}
	}

	for _, wb := range pc.wbQueue {
		wb.snap = false
	}

	for word, set := range pc.snapSlots {
		for set != 0 {
			bit := bits.TrailingZeros64(set)
			set &= set - 1

			pc.protected.set(uint32(word*wordBits + bit)) //nolint:gosec // Slot counts fit in uint32.
		}
	}

	pc.snapTable = nil
	pc.snapSlots = nil
}

// stop tells the background writer to exit once its queue is empty.
func (pc *pageCache) stop() {
	pc.wbStopped = true
	pc.epoch++
	pc.wbCond.Broadcast()
}

// reset forgets every page without writing anything; slots the durable
// checkpoint uses stay protected. The background writer must be idle.
func (pc *pageCache) reset() {
	for i := range pc.framePage {
		pc.framePage[i] = noPage
		pc.flags[i] = 0
		pc.pins[i] = 0
	}

	for _, wb := range pc.wbQueue {
		pc.recycle(wb.buf)
	}

	clear(pc.wb)
	pc.wbQueue = pc.wbQueue[:0]
	pc.frameOf = pc.frameOf[:0]
	pc.version = pc.version[:0]
	pc.pageTable = pc.pageTable[:0]
	pc.freeSlots = pc.freeSlots[:0]

	for slot := range pc.physSlots {
		if !pc.protected.has(slot) {
			pc.freeSlots = append(pc.freeSlots, slot)
		}
	}

	pc.hand = 0
	pc.epoch++
}
