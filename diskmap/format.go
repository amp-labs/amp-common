package diskmap

import (
	"encoding/binary"
	"hash/crc32"
	"math/bits"
)

// On-disk layout.
//
// Values live in append-only segment files. Each record is:
//
//	[crc32c uint32][key length uint32][value length and flags uint32]
//	[sequence uint64][key][value][move trailer, 16 bytes, moves only]
//
// The checksum covers everything after itself. The top two bits of the value
// length mark tombstones (written by Remove) and moves (written by
// compaction, which copies a live record and notes where it came from). Puts
// and tombstones are numbered by one counter, so replaying them in order
// rebuilds the map exactly; moves have their own counter and record the put
// sequence they followed. Records never change once written; compaction
// copies live ones elsewhere and deletes the old file.
//
// Each shard's index is an extendible hash table stored in a file of
// fixed-size pages. A page is one bucket:
//
//	[count uint16][local depth uint8][5 bytes reserved]
//	[slotsPerPage tags, one byte each, padded to a whole word]
//	[slotsPerPage hashes, uint64 each]
//	[slotsPerPage locations, 12 bytes each: segment, offset, record size]
//
// A tag is one byte of the hash. Lookups compare a word of tags at a time and
// read a full hash only where a tag matches, so a lookup touches about four
// cache lines of a page instead of scanning every hash.
const (
	pageSize     = 4096
	pageHeader   = 8
	tagsPerWord  = 8
	hashSize     = 8
	locSize      = 12
	slotsPerPage = (pageSize - pageHeader - tagsPerWord) / (1 + hashSize + locSize)
	tagsOffset   = pageHeader
	hashesOffset = tagsOffset + (slotsPerPage+tagsPerWord-1)/tagsPerWord*tagsPerWord
	locsOffset   = hashesOffset + slotsPerPage*hashSize

	// hashBits is the width of a key hash. The directory uses its top bits,
	// shards its bottom bits and tags the byte at tagShift.
	hashBits = 64
	tagShift = 8

	recordHeaderSize = 20
	moveTrailerSize  = 16

	flagTombstone = 1 << 31
	flagMove      = 1 << 30
	lengthMask    = flagMove - 1

	// MaxKeySize is the largest key the map accepts, in bytes.
	MaxKeySize = 1<<16 - 1

	// MaxValueSize is the largest value the map accepts, in bytes.
	MaxValueSize = 1<<30 - 1
)

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// loc is where a record lives: segment id, byte offset and total record size.
type loc struct {
	seg  uint32
	off  uint32
	size uint32
}

// recordKind tells puts, tombstones and moves apart.
type recordKind int

const (
	kindPut recordKind = iota
	kindTombstone
	kindMove
)

// moveInfo is the trailer of a move record: the put sequence the move
// followed and the location the record was copied from.
type moveInfo struct {
	at   uint64
	from loc
}

func recordSize(keyLen, valueLen int) int {
	return recordHeaderSize + keyLen + valueLen
}

// appendRecord encodes a put (or, with tombstone set, a tombstone) onto dst.
func appendRecord(dst []byte, seq uint64, key string, value []byte, tombstone bool) []byte {
	start := len(dst)

	lengthAndFlags := uint32(len(value)) //nolint:gosec // Bounded by MaxValueSize.
	if tombstone {
		lengthAndFlags |= flagTombstone
	}

	dst = binary.LittleEndian.AppendUint32(dst, 0)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(key))) //nolint:gosec // Bounded by MaxKeySize.
	dst = binary.LittleEndian.AppendUint32(dst, lengthAndFlags)
	dst = binary.LittleEndian.AppendUint64(dst, seq)
	dst = append(dst, key...)
	dst = append(dst, value...)

	binary.LittleEndian.PutUint32(dst[start:], crc32.Checksum(dst[start+4:], castagnoli))

	return dst
}

// appendMove encodes a copy of the live record rec, found at from, as a move
// record onto dst.
func appendMove(dst []byte, rec []byte, seq uint64, info moveInfo) []byte {
	start := len(dst)
	keyLen, valueLen := recordLengths(rec)
	body := rec[recordHeaderSize : recordHeaderSize+keyLen+valueLen]

	dst = binary.LittleEndian.AppendUint32(dst, 0)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(keyLen))            //nolint:gosec // Bounded by MaxKeySize.
	dst = binary.LittleEndian.AppendUint32(dst, uint32(valueLen)|flagMove) //nolint:gosec // Bounded by MaxValueSize.
	dst = binary.LittleEndian.AppendUint64(dst, seq)
	dst = append(dst, body...)
	dst = binary.LittleEndian.AppendUint64(dst, info.at)
	dst = binary.LittleEndian.AppendUint32(dst, info.from.seg)
	dst = binary.LittleEndian.AppendUint32(dst, info.from.off)

	binary.LittleEndian.PutUint32(dst[start:], crc32.Checksum(dst[start+4:], castagnoli))

	return dst
}

// recordLengths reads the key and value lengths from a record header.
func recordLengths(rec []byte) (keyLen, valueLen int) {
	return int(binary.LittleEndian.Uint32(rec[4:])), int(binary.LittleEndian.Uint32(rec[8:]) & lengthMask)
}

func recordKindOf(rec []byte) recordKind {
	switch flags := binary.LittleEndian.Uint32(rec[8:]); {
	case flags&flagTombstone != 0:
		return kindTombstone
	case flags&flagMove != 0:
		return kindMove
	default:
		return kindPut
	}
}

func recordSeq(rec []byte) uint64 {
	return binary.LittleEndian.Uint64(rec[12:])
}

// fullRecordSize returns the size of the record whose header starts rec.
func fullRecordSize(rec []byte) int {
	keyLen, valueLen := recordLengths(rec)

	size := recordSize(keyLen, valueLen)
	if recordKindOf(rec) == kindMove {
		size += moveTrailerSize
	}

	return size
}

// recordKey returns the key of a record whose header is complete, or nil if
// the record is too short to hold it.
func recordKey(rec []byte) []byte {
	keyLen, _ := recordLengths(rec)
	if recordHeaderSize+keyLen > len(rec) {
		return nil
	}

	return rec[recordHeaderSize : recordHeaderSize+keyLen]
}

// recordMove reads the trailer of a complete move record.
func recordMove(rec []byte) moveInfo {
	trailer := rec[len(rec)-moveTrailerSize:]

	return moveInfo{
		at: binary.LittleEndian.Uint64(trailer),
		from: loc{
			seg: binary.LittleEndian.Uint32(trailer[8:]),
			off: binary.LittleEndian.Uint32(trailer[12:]),
		},
	}
}

// validRecord reports whether rec holds exactly one complete record with a
// correct checksum.
func validRecord(rec []byte) bool {
	return len(rec) >= recordHeaderSize && fullRecordSize(rec) == len(rec) &&
		crc32.Checksum(rec[4:], castagnoli) == binary.LittleEndian.Uint32(rec)
}

// parseRecord splits a complete record into key and value, verifying its
// length and, when verify is set, its checksum.
func parseRecord(rec []byte, verify bool) (key, value []byte, err error) {
	if len(rec) < recordHeaderSize || fullRecordSize(rec) != len(rec) {
		return nil, nil, ErrCorrupt
	}

	if verify && crc32.Checksum(rec[4:], castagnoli) != binary.LittleEndian.Uint32(rec) {
		return nil, nil, ErrCorrupt
	}

	keyLen, valueLen := recordLengths(rec)
	key = rec[recordHeaderSize : recordHeaderSize+keyLen]

	return key, rec[recordHeaderSize+keyLen : recordHeaderSize+keyLen+valueLen], nil
}

func pageCount(p []byte) int {
	return int(binary.LittleEndian.Uint16(p))
}

func setPageCount(p []byte, n int) {
	binary.LittleEndian.PutUint16(p, uint16(n)) //nolint:gosec // At most slotsPerPage.
}

func pageDepth(p []byte) uint8 {
	return p[2]
}

func setPageDepth(p []byte, depth uint8) {
	p[2] = depth
}

func slotHash(p []byte, i int) uint64 {
	return binary.LittleEndian.Uint64(p[hashesOffset+i*hashSize:])
}

// tagOf picks the hash byte stored as a slot's tag. The low bits choose the
// shard and the high bits the directory entry, so this byte varies within a
// page.
func tagOf(h uint64) byte {
	return byte(h >> tagShift) //nolint:gosec // Deliberate truncation.
}

func slotLoc(p []byte, i int) loc {
	b := p[locsOffset+i*locSize:]

	return loc{
		seg:  binary.LittleEndian.Uint32(b),
		off:  binary.LittleEndian.Uint32(b[4:]),
		size: binary.LittleEndian.Uint32(b[8:]),
	}
}

func setSlotLoc(p []byte, i int, l loc) {
	b := p[locsOffset+i*locSize:]

	binary.LittleEndian.PutUint32(b, l.seg)
	binary.LittleEndian.PutUint32(b[4:], l.off)
	binary.LittleEndian.PutUint32(b[8:], l.size)
}

func setSlot(p []byte, i int, h uint64, l loc) {
	p[tagsOffset+i] = tagOf(h)
	binary.LittleEndian.PutUint64(p[hashesOffset+i*hashSize:], h)
	setSlotLoc(p, i, l)
}

// removeSlot deletes slot i by moving the last slot into its place.
func removeSlot(p []byte, i int) {
	last := pageCount(p) - 1
	if i != last {
		setSlot(p, i, slotHash(p, last), slotLoc(p, last))
	}

	setPageCount(p, last)
}

const (
	lowBits  = 0x0101010101010101
	highBits = 0x8080808080808080
)

// slotMatcher yields the slots of a page whose hash equals h, using the tags
// to skip most slots a word at a time.
type slotMatcher struct {
	page []byte
	hash uint64
	want uint64 // the tag repeated in every byte
	n    int
	base int
	hits uint64
}

func matchSlots(p []byte, h uint64) slotMatcher {
	return slotMatcher{page: p, hash: h, want: uint64(tagOf(h)) * lowBits, n: pageCount(p), base: -tagsPerWord}
}

// next returns the next matching slot, or -1.
func (s *slotMatcher) next() int {
	for {
		for s.hits != 0 {
			i := s.base + bits.TrailingZeros64(s.hits)/tagsPerWord
			s.hits &= s.hits - 1

			if slotHash(s.page, i) == s.hash {
				return i
			}
		}

		s.base += tagsPerWord
		if s.base >= s.n {
			return -1
		}

		// Bytes of x are zero where tags match. The bit trick can also flag a
		// byte just above a real match; the full hash check discards those.
		x := binary.LittleEndian.Uint64(s.page[tagsOffset+s.base:]) ^ s.want
		s.hits = (x - lowBits) &^ x & highBits

		if left := s.n - s.base; left < tagsPerWord {
			s.hits &= 1<<(tagsPerWord*left) - 1
		}
	}
}
