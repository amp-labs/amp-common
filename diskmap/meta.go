package diskmap

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	metaFile  = "meta"
	indexFile = "index"

	metaMagic   = "DISKMAP1"
	metaVersion = 1

	filePerm = 0o600
	dirPerm  = 0o750
)

// segmentMeta is the persisted accounting for one segment.
type segmentMeta struct {
	id   uint32
	size int64
	dead int64
}

// meta is everything needed to reopen a cleanly closed map. It exists only
// between Close and the next Open, so a directory with data files but no meta
// file was not closed cleanly.
type meta struct {
	seed     uint64
	count    uint64
	numPages uint32
	depth    uint8
	nextSeg  uint32
	segments []segmentMeta // ascending id; the last one is the active segment
	dir      []uint32
}

func (m *meta) encode() []byte {
	buf := make([]byte, 0, 64+len(m.segments)*20+len(m.dir)*4)

	buf = append(buf, metaMagic...)
	buf = binary.LittleEndian.AppendUint32(buf, metaVersion)
	buf = binary.LittleEndian.AppendUint64(buf, m.seed)
	buf = binary.LittleEndian.AppendUint64(buf, m.count)
	buf = binary.LittleEndian.AppendUint32(buf, m.numPages)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(m.depth))
	buf = binary.LittleEndian.AppendUint32(buf, m.nextSeg)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.segments))) //nolint:gosec // Segment ids are uint32.

	for _, s := range m.segments {
		buf = binary.LittleEndian.AppendUint32(buf, s.id)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(s.size)) //nolint:gosec // Sizes are non-negative.
		buf = binary.LittleEndian.AppendUint64(buf, uint64(s.dead)) //nolint:gosec // Sizes are non-negative.
	}

	for _, id := range m.dir {
		buf = binary.LittleEndian.AppendUint32(buf, id)
	}

	return binary.LittleEndian.AppendUint32(buf, crc32.Checksum(buf, castagnoli))
}

// metaReader decodes fixed-width fields and remembers the first short read.
type metaReader struct {
	buf []byte
	err error
}

func (r *metaReader) take(n int) []byte {
	if r.err != nil || len(r.buf) < n {
		r.err = ErrCorrupt

		return make([]byte, n)
	}

	b := r.buf[:n]
	r.buf = r.buf[n:]

	return b
}

func (r *metaReader) u32() uint32 { return binary.LittleEndian.Uint32(r.take(4)) }
func (r *metaReader) u64() uint64 { return binary.LittleEndian.Uint64(r.take(8)) }

func decodeMeta(buf []byte) (*meta, error) {
	if len(buf) < len(metaMagic)+4 {
		return nil, fmt.Errorf("%w: meta file too short", ErrCorrupt)
	}

	body, sum := buf[:len(buf)-4], binary.LittleEndian.Uint32(buf[len(buf)-4:])
	if crc32.Checksum(body, castagnoli) != sum {
		return nil, fmt.Errorf("%w: meta checksum mismatch", ErrCorrupt)
	}

	r := &metaReader{buf: body}

	if string(r.take(len(metaMagic))) != metaMagic {
		return nil, fmt.Errorf("%w: not a diskmap directory", ErrCorrupt)
	}

	if v := r.u32(); v != metaVersion {
		return nil, fmt.Errorf("%w: unsupported format version %d", ErrCorrupt, v)
	}

	m := &meta{
		seed:     r.u64(),
		count:    r.u64(),
		numPages: r.u32(),
	}

	depth := r.u32()
	m.nextSeg = r.u32()
	nsegs := r.u32()

	if depth > maxDepth || nsegs == 0 || int(nsegs) > len(r.buf)/20 {
		return nil, fmt.Errorf("%w: bad meta header", ErrCorrupt)
	}

	m.depth = uint8(depth)

	for range nsegs {
		m.segments = append(m.segments, segmentMeta{
			id:   r.u32(),
			size: int64(r.u64()), //nolint:gosec // Written from non-negative int64.
			dead: int64(r.u64()), //nolint:gosec // Written from non-negative int64.
		})
	}

	m.dir = make([]uint32, 1<<depth)
	for i := range m.dir {
		m.dir[i] = r.u32()
		if m.dir[i] >= m.numPages {
			return nil, fmt.Errorf("%w: directory points past the index", ErrCorrupt)
		}
	}

	if r.err != nil || len(r.buf) != 0 {
		return nil, fmt.Errorf("%w: bad meta length", ErrCorrupt)
	}

	return m, nil
}

// readMeta loads the meta file, returning nil if there is none.
func readMeta(dir string) (*meta, error) {
	buf, err := os.ReadFile(filepath.Join(dir, metaFile)) //nolint:gosec // Path built from dir.
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("diskmap: reading meta: %w", err)
	}

	return decodeMeta(buf)
}

// writeMeta durably replaces the meta file.
func writeMeta(dir string, m *meta) error {
	tmp := filepath.Join(dir, metaFile+".tmp")

	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm) //nolint:gosec // Path built from dir.
	if err != nil {
		return fmt.Errorf("diskmap: writing meta: %w", err)
	}

	_, err = file.Write(m.encode())
	if err == nil {
		err = file.Sync()
	}

	if closeErr := file.Close(); err == nil {
		err = closeErr
	}

	if err == nil {
		err = os.Rename(tmp, filepath.Join(dir, metaFile))
	}

	if err == nil {
		err = syncDir(dir)
	}

	if err != nil {
		return fmt.Errorf("diskmap: writing meta: %w", err)
	}

	return nil
}

// removeMeta deletes the meta file, marking the directory as in use.
func removeMeta(dir string) error {
	if err := os.Remove(filepath.Join(dir, metaFile)); err != nil {
		return fmt.Errorf("diskmap: removing meta: %w", err)
	}

	return syncDir(dir)
}

func syncDir(dir string) error {
	d, err := os.Open(dir) //nolint:gosec // Caller-provided directory.
	if err != nil {
		return err
	}

	err = d.Sync()
	if closeErr := d.Close(); err == nil {
		err = closeErr
	}

	return err
}

// dataFiles lists the files in dir that belong to a map.
func dataFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("diskmap: reading directory: %w", err)
	}

	var files []string

	for _, e := range entries {
		name := e.Name()
		if name == metaFile || name == metaFile+".tmp" || name == indexFile || isSegmentName(name) {
			files = append(files, name)
		}
	}

	slices.Sort(files)

	return files, nil
}

func isSegmentName(name string) bool {
	stem, ok := strings.CutSuffix(name, segmentSuffix)
	if !ok {
		return false
	}

	_, err := strconv.ParseUint(stem, 10, 32)

	return err == nil
}

// removeDataFiles deletes every file in dir that belongs to a map.
func removeDataFiles(dir string) error {
	files, err := dataFiles(dir)
	if err != nil {
		return err
	}

	for _, name := range files {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("diskmap: removing %s: %w", name, err)
		}
	}

	return nil
}
