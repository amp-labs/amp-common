package diskmap

import (
	"encoding/binary"
)

const (
	vcHeader  = 16 // record location, key length, value length
	vcSkip    = ^uint32(0)
	doorBits  = 10
	doorSlots = 1 << doorBits

	// fibonacciHash multiplies a key to spread it over the doorkeeper slots.
	fibonacciHash = 0x9e3779b97f4a7c15

	// Records bigger than the ring divided by maxEntryShare are not cached,
	// so one large value cannot flush everything else.
	maxEntryShare = 8

	// One byte in indexShare of the cache's budget pays for its index map.
	indexShare = 4

	// vcMapEntryCost approximates the memory a Go map spends per entry; the
	// entry count is capped so the map stays within its share of the budget.
	vcMapEntryCost = 24
)

// valueCache keeps recently read records in memory. It is keyed by record
// location rather than by key: a record never changes and a location is
// never reused, so an entry can never be stale. Overwritten or removed
// entries simply stop being looked up and age out.
//
// Entries are stored in a ring buffer that is evicted oldest first, so all
// the bytes live in one pointer-free allocation the garbage collector does
// not scan. A record is admitted on its second miss, tracked by a small
// direct-mapped doorkeeper, so a stream of one-off reads does not churn the
// cache. A valueCache belongs to one shard and is guarded by its lock.
type valueCache struct {
	ring       []byte
	head       uint64 // ring position of the next entry
	tail       uint64 // ring position of the oldest entry
	index      map[uint64]uint64
	maxEntries int
	door       [doorSlots]uint64

	hits   uint64
	misses uint64
}

func newValueCache(bytes int) *valueCache {
	ringBytes := bytes - bytes/indexShare

	return &valueCache{
		ring:       make([]byte, ringBytes),
		index:      make(map[uint64]uint64),
		maxEntries: max((bytes-ringBytes)/vcMapEntryCost, 1),
	}
}

func locKey(l loc) uint64 {
	return uint64(l.seg)<<32 | uint64(l.off)
}

// get returns the value of the record at l if it is cached and holds key.
// The slice aliases the cache and is valid only while the lock is held.
func (vc *valueCache) get(l loc, key string) (value []byte, cached bool) {
	pos, ok := vc.index[locKey(l)]
	if !ok {
		vc.misses++

		return nil, false
	}

	entry := vc.ring[pos%uint64(len(vc.ring)):]
	keyLen := int(binary.LittleEndian.Uint32(entry[8:]))
	valueLen := int(binary.LittleEndian.Uint32(entry[12:]))

	// A cached record for another key with the same hash is a hit for the
	// record but not for this key.
	if string(entry[vcHeader:vcHeader+keyLen]) != key {
		return nil, true
	}

	vc.hits++

	return entry[vcHeader+keyLen : vcHeader+keyLen+valueLen], true
}

// admit reports whether a record that just missed should be cached: true on
// its second miss in a row through the same doorkeeper slot.
func (vc *valueCache) admit(l loc, size int) bool {
	if size+vcHeader > len(vc.ring)/maxEntryShare {
		return false
	}

	entryKey := locKey(l)
	slot := &vc.door[(entryKey*fibonacciHash)>>(hashBits-doorBits)]

	if *slot == entryKey {
		*slot = 0

		return true
	}

	*slot = entryKey

	return false
}

// put caches the record at l.
func (vc *valueCache) put(l loc, key string, value []byte) {
	entryKey := locKey(l)

	_, ok := vc.index[entryKey]
	if ok {
		return
	}

	size := uint64(vcHeader + len(key) + len(value))
	ringLen := uint64(len(vc.ring))

	if size > ringLen/maxEntryShare {
		return
	}

	// Entries never wrap: if this one would cross the end of the ring, skip
	// to the start.
	if off := vc.head % ringLen; off+size > ringLen {
		vc.makeRoom(ringLen - off)
		vc.markSkip(vc.head)
		vc.head += ringLen - off
	}

	vc.makeRoom(size)

	for len(vc.index) >= vc.maxEntries {
		vc.evictOldest()
	}

	entry := vc.ring[vc.head%ringLen:]
	binary.LittleEndian.PutUint64(entry, entryKey)
	binary.LittleEndian.PutUint32(entry[8:], uint32(len(key)))    //nolint:gosec // Bounded by MaxKeySize.
	binary.LittleEndian.PutUint32(entry[12:], uint32(len(value))) //nolint:gosec // Bounded by the ring.
	copy(entry[vcHeader:], key)
	copy(entry[vcHeader+len(key):], value)

	vc.index[entryKey] = vc.head
	vc.head += size
}

// makeRoom evicts the oldest entries until n bytes after head are free.
func (vc *valueCache) makeRoom(n uint64) {
	for vc.head+n-vc.tail > uint64(len(vc.ring)) {
		vc.evictOldest()
	}
}

func (vc *valueCache) markSkip(pos uint64) {
	ringLen := uint64(len(vc.ring))
	if off := pos % ringLen; ringLen-off >= vcHeader {
		binary.LittleEndian.PutUint32(vc.ring[off+12:], vcSkip)
	}
}

// evictOldest drops the entry at the tail.
func (vc *valueCache) evictOldest() {
	ringLen := uint64(len(vc.ring))
	off := vc.tail % ringLen

	// Fewer than a header's worth of bytes before the end is always a skip.
	if ringLen-off < vcHeader {
		vc.tail += ringLen - off

		return
	}

	entry := vc.ring[off:]

	valueLen := binary.LittleEndian.Uint32(entry[12:])
	if valueLen == vcSkip {
		vc.tail += ringLen - off

		return
	}

	k := binary.LittleEndian.Uint64(entry)
	if pos, ok := vc.index[k]; ok && pos == vc.tail {
		delete(vc.index, k)
	}

	vc.tail += uint64(vcHeader) + uint64(binary.LittleEndian.Uint32(entry[8:])) + uint64(valueLen)
}

func (vc *valueCache) reset() {
	clear(vc.index)
	clear(vc.door[:])
	vc.head, vc.tail = 0, 0
}
