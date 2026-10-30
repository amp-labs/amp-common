package diskmap

import (
	"encoding/binary"
	"hash/crc32"
)

// On-disk layout.
//
// Values live in append-only segment files. Each record is:
//
//	[crc32c uint32][key length uint32][value length uint32][key][value]
//
// The checksum covers everything after itself. Records never move inside a
// segment; compaction copies live records to the active segment and deletes
// the old file.
//
// The index is an extendible hash table stored in one file of fixed-size
// pages. A page is one bucket:
//
//	[count uint16][local depth uint8][5 bytes reserved]
//	[slotsPerPage hashes, uint64 each]
//	[slotsPerPage locations, 12 bytes each: segment, offset, record size]
//
// Hashes and locations are stored as two arrays so a lookup scans a dense run
// of hashes and touches a location only on a match.
const (
	pageSize     = 4096
	pageHeader   = 8
	hashSize     = 8
	locSize      = 12
	slotsPerPage = (pageSize - pageHeader) / (hashSize + locSize)
	locsOffset   = pageHeader + slotsPerPage*hashSize

	recordHeaderSize = 12

	// MaxKeySize is the largest key the map accepts, in bytes.
	MaxKeySize = 1<<16 - 1

	// MaxValueSize is the largest value the map accepts, in bytes.
	MaxValueSize = 1 << 30
)

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// loc is where a record lives: segment id, byte offset and total record size.
type loc struct {
	seg  uint32
	off  uint32
	size uint32
}

func recordSize(keyLen, valueLen int) int {
	return recordHeaderSize + keyLen + valueLen
}

// appendRecord encodes a record onto dst.
func appendRecord(dst []byte, key string, value []byte) []byte {
	start := len(dst)

	dst = binary.LittleEndian.AppendUint32(dst, 0)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(key)))   //nolint:gosec // Bounded by MaxKeySize.
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(value))) //nolint:gosec // Bounded by MaxValueSize.
	dst = append(dst, key...)
	dst = append(dst, value...)

	binary.LittleEndian.PutUint32(dst[start:], crc32.Checksum(dst[start+4:], castagnoli))

	return dst
}

// recordLengths reads the key and value lengths from a record header.
func recordLengths(rec []byte) (keyLen, valueLen int) {
	return int(binary.LittleEndian.Uint32(rec[4:])), int(binary.LittleEndian.Uint32(rec[8:]))
}

// parseRecord splits a complete record into key and value, verifying its
// length and, when verify is set, its checksum.
func parseRecord(rec []byte, verify bool) (key, value []byte, err error) {
	if len(rec) < recordHeaderSize {
		return nil, nil, ErrCorrupt
	}

	keyLen, valueLen := recordLengths(rec)
	if recordSize(keyLen, valueLen) != len(rec) {
		return nil, nil, ErrCorrupt
	}

	if verify && crc32.Checksum(rec[4:], castagnoli) != binary.LittleEndian.Uint32(rec) {
		return nil, nil, ErrCorrupt
	}

	key = rec[recordHeaderSize : recordHeaderSize+keyLen]

	return key, rec[recordHeaderSize+keyLen:], nil
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
	return binary.LittleEndian.Uint64(p[pageHeader+i*hashSize:])
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
	binary.LittleEndian.PutUint64(p[pageHeader+i*hashSize:], h)
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
