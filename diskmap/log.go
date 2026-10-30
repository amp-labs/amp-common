package diskmap

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// valueLog is the append-only store of records, shared by all shards. Its
// lock guards the write buffers and segment accounting; reads of records that
// are already in a file need no lock at all.
//
// Writes are double buffered. New records go into cur; when it fills, it is
// handed to the flusher goroutine as pending and the other buffer takes its
// place, so writers only wait if the flusher falls a whole buffer behind.
// Records in either buffer are served to readers from memory.
type valueLog struct {
	mu sync.Mutex

	dir     string
	segSize int64
	table   atomic.Pointer[segTable]
	nextSeg uint32

	cur       memBuf // cur.seg is the active segment
	pending   memBuf // being written by the flusher; data == nil when idle
	spare     []byte
	bufCap    int
	flushCond *sync.Cond

	total int64 // bytes in all segments
	dead  int64 // bytes of overwritten, removed or relocated records

	// gen counts Clear calls. A record appended in one generation must not
	// be indexed in another. It changes only while every shard lock and the
	// log lock are held.
	gen atomic.Uint64

	// iterators counts Range calls in progress; compaction pauses while it is
	// nonzero.
	iterators atomic.Int32

	// writeMu is held for reading from the moment a record is appended until
	// the index points at it (by Add, and by the compactor for each batch it
	// relocates), and for writing while Range takes its snapshot. A Range
	// therefore never sees a record that is appended but not yet indexed,
	// which would let it visit an overwritten key twice.
	writeMu sync.RWMutex

	// Compaction coordination; see compact.go.
	compactCond  *sync.Cond
	compactDone  *sync.Cond
	compactIdle  bool
	forceCompact bool
	compacted    int64
	threshold    float64

	closed   bool // no more appends
	stopping bool // background goroutines should exit
	failed   atomic.Pointer[error]
}

// memBuf holds records of one segment that are not yet in its file.
type memBuf struct {
	seg   *segment
	start int64 // segment offset of data[0]
	data  []byte
}

func (b *memBuf) end() int64 {
	return b.start + int64(len(b.data))
}

func newValueLog(dir string, segSize int64, bufCap int, threshold float64) *valueLog {
	lg := &valueLog{
		dir:       dir,
		segSize:   segSize,
		bufCap:    bufCap,
		spare:     make([]byte, 0, bufCap),
		threshold: threshold,
	}
	lg.table.Store(&segTable{})
	lg.flushCond = sync.NewCond(&lg.mu)
	lg.compactCond = sync.NewCond(&lg.mu)
	lg.compactDone = sync.NewCond(&lg.mu)

	return lg
}

// load opens the segments of a saved map.
func (lg *valueLog) load(saved *meta) error {
	live := make([]*segment, 0, len(saved.segments))

	for _, sm := range saved.segments {
		s, err := openSegment(lg.dir, sm.id, sm.size, sm.dead)
		if err != nil {
			for _, s := range live {
				s.release()
			}

			return err
		}

		live = append(live, s)
		lg.total += s.size
		lg.dead += s.dead
	}

	lg.table.Store(newSegTable(live))
	lg.nextSeg = saved.nextSeg

	active := live[len(live)-1]
	if err := active.file.Truncate(active.size); err != nil {
		return fmt.Errorf("diskmap: truncating active segment: %w", err)
	}

	lg.cur = memBuf{seg: active, start: active.size, data: make([]byte, 0, lg.bufCap)}

	return nil
}

func (lg *valueLog) segment(id uint32) *segment {
	return lg.table.Load().get(id)
}

func (lg *valueLog) err() error {
	if p := lg.failed.Load(); p != nil {
		return *p
	}

	return nil
}

// fail marks the map unusable after a failed write left its files
// inconsistent. The caller must hold lg.mu.
func (lg *valueLog) fail(err error) error {
	if lg.failed.CompareAndSwap(nil, &err) {
		lg.flushCond.Broadcast()
		lg.compactCond.Broadcast()
		lg.compactDone.Broadcast()
	}

	return err
}

// appendRecord writes a record for key and value and returns its location
// and the generation it was written in.
func (lg *valueLog) appendRecord(key string, value []byte) (loc, uint64, error) {
	size := recordSize(len(key), len(value))

	lg.mu.Lock()
	defer lg.mu.Unlock()

	if err := lg.ensureRoom(size); err != nil {
		return loc{}, 0, err
	}

	if size > lg.bufCap {
		return lg.appendOversized(appendRecord(make([]byte, 0, size), key, value)), lg.gen.Load(), nil
	}

	l := lg.appended(size)
	lg.cur.data = appendRecord(lg.cur.data, key, value)

	return l, lg.gen.Load(), nil
}

// appendRaw copies an encoded record into the log.
func (lg *valueLog) appendRaw(rec []byte) (loc, uint64, error) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	if err := lg.ensureRoom(len(rec)); err != nil {
		return loc{}, 0, err
	}

	if len(rec) > lg.bufCap {
		return lg.appendOversized(rec), lg.gen.Load(), nil
	}

	l := lg.appended(len(rec))
	lg.cur.data = append(lg.cur.data, rec...)

	return l, lg.gen.Load(), nil
}

// ensureRoom prepares the active segment for a record of the given size:
// it starts a new segment if this one would grow past the segment size, and
// hands cur to the flusher if the record does not fit, waiting for the
// flusher if it is busy. On success the record fits in cur, or cur is empty
// and the flusher idle (for records larger than a buffer).
func (lg *valueLog) ensureRoom(size int) error {
	for {
		if err := lg.err(); err != nil {
			return err
		}

		if lg.closed {
			return ErrClosed
		}

		if s := lg.cur.seg; s.size > 0 && s.size+int64(size) > lg.segSize {
			if len(lg.cur.data) > 0 {
				if lg.pending.data != nil {
					lg.flushCond.Wait()

					continue
				}

				lg.handOff()
			}

			if err := lg.rotate(); err != nil {
				return err
			}

			continue
		}

		if len(lg.cur.data)+size <= lg.bufCap {
			return nil
		}

		if lg.pending.data != nil {
			lg.flushCond.Wait()

			continue
		}

		if len(lg.cur.data) == 0 {
			return nil
		}

		lg.handOff()
	}
}

// appendOversized hands a record larger than a write buffer straight to the
// idle flusher. It takes ownership of rec.
func (lg *valueLog) appendOversized(rec []byte) loc {
	l := lg.appended(len(rec))
	lg.pending = memBuf{seg: lg.cur.seg, start: lg.cur.start, data: rec}
	lg.cur.start += int64(len(rec))
	lg.flushCond.Broadcast()

	return l
}

// appended accounts for size bytes about to be added at the end of the active
// segment and returns their location.
func (lg *valueLog) appended(size int) loc {
	s := lg.cur.seg
	l := loc{seg: s.id, off: uint32(s.size), size: uint32(size)} //nolint:gosec // Segments stay below 4 GiB.

	s.size += int64(size)
	s.unsynced = true
	lg.total += int64(size)

	return l
}

// handOff gives cur to the idle flusher and starts an empty cur after it.
func (lg *valueLog) handOff() {
	lg.pending = lg.cur

	spare := lg.spare
	if spare == nil {
		spare = make([]byte, 0, lg.bufCap)
	}

	lg.cur = memBuf{seg: lg.pending.seg, start: lg.pending.end(), data: spare}
	lg.spare = nil
	lg.flushCond.Broadcast()
}

// rotate starts a new active segment. cur must be empty.
func (lg *valueLog) rotate() error {
	s, err := createSegment(lg.dir, lg.nextSeg)
	if err != nil {
		return lg.fail(err)
	}

	lg.nextSeg++
	lg.table.Store(lg.table.Load().with(s))

	data := lg.cur.data
	if data == nil {
		data = make([]byte, 0, lg.bufCap)
	}

	lg.cur = memBuf{seg: s, data: data[:0]}
	lg.wakeCompactor(nil)

	return nil
}

// drain waits until every buffered record is in its file.
func (lg *valueLog) drain() error {
	for {
		if err := lg.err(); err != nil {
			return err
		}

		if lg.pending.data != nil {
			lg.flushCond.Wait()

			continue
		}

		if len(lg.cur.data) == 0 {
			return nil
		}

		lg.handOff()
	}
}

// flushLoop writes pending buffers to their segment files.
func (lg *valueLog) flushLoop(wg *sync.WaitGroup) {
	defer wg.Done()

	lg.mu.Lock()
	defer lg.mu.Unlock()

	for {
		for lg.pending.data == nil {
			if lg.stopping {
				return
			}

			lg.flushCond.Wait()
		}

		p := lg.pending

		lg.mu.Unlock()
		_, err := p.seg.file.WriteAt(p.data, p.start)
		lg.mu.Lock()

		if err != nil {
			lg.fail(fmt.Errorf("diskmap: writing segment %d: %w", p.seg.id, err))
		} else {
			p.seg.written.Store(p.end())
		}

		// One-off buffers for oversized records are not recycled.
		if cap(p.data) == lg.bufCap && lg.spare == nil {
			lg.spare = p.data[:0]
		}

		lg.pending = memBuf{}
		lg.flushCond.Broadcast()
	}
}

// markDead records that size bytes of segment id no longer hold a live
// record.
func (lg *valueLog) markDead(id uint32, size uint32) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	lg.markDeadLocked(id, int64(size))
}

func (lg *valueLog) markDeadLocked(id uint32, size int64) {
	s := lg.segment(id)
	if s == nil {
		return
	}

	s.dead += size
	lg.dead += size
	lg.wakeCompactor(s)
}

// read fills buf with the bytes of segment s starting at off. The caller
// must hold a reference to s or otherwise keep it from being dropped. Bytes
// already in the file are read without any lock; the newest bytes may still
// be in a write buffer, which takes the log lock to copy.
func (lg *valueLog) read(s *segment, off int64, buf []byte) error {
	if off+int64(len(buf)) > s.written.Load() {
		lg.mu.Lock()
		buf = buf[:lg.copyBuffered(s, off, buf)]
		lg.mu.Unlock()
	}

	if len(buf) == 0 {
		return nil
	}

	return readFile(s, off, buf)
}

// copyBuffered copies the parts of [off, off+len(buf)) of segment s that are
// still in a write buffer and returns how many leading bytes of buf must be
// read from the file instead. The caller must hold lg.mu.
func (lg *valueLog) copyBuffered(s *segment, off int64, buf []byte) int {
	end := off + int64(len(buf))

	for _, b := range [...]*memBuf{&lg.pending, &lg.cur} {
		if b.seg != s {
			continue
		}

		if lo, hi := max(off, b.start), min(end, b.end()); lo < hi {
			copy(buf[lo-off:hi-off], b.data[lo-b.start:hi-b.start])
		}
	}

	return int(min(max(s.written.Load()-off, 0), int64(len(buf))))
}
