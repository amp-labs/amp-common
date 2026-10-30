package diskmap

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

const (
	metaFile = "meta"

	metaMagic   = "DISKMAP1"
	metaVersion = 1

	u32Size        = 4
	u64Size        = 8
	segmentMetaLen = u32Size + 2*u64Size

	filePerm = 0o600
	dirPerm  = 0o750
)

// segmentMeta is the persisted accounting for one segment.
type segmentMeta struct {
	id   uint32
	size int64
	dead int64
}

// shardMeta is the persisted state of one shard's index.
type shardMeta struct {
	count    uint64
	numPages uint32
	depth    uint8
	dir      []uint32
}

// meta is everything needed to reopen a cleanly closed map. It exists only
// between Close and the next Open, so a directory with data files but no meta
// file was not closed cleanly.
type meta struct {
	seed     uint64
	nextSeg  uint32
	segments []segmentMeta // ascending id; the last one is the active segment
	shards   []shardMeta
}

func (m *meta) encode() []byte {
	buf := make([]byte, 0, len(metaMagic)+len(m.segments)*segmentMetaLen+len(m.shards)*segmentMetaLen)

	buf = append(buf, metaMagic...)
	buf = binary.LittleEndian.AppendUint32(buf, metaVersion)
	buf = binary.LittleEndian.AppendUint64(buf, m.seed)
	buf = binary.LittleEndian.AppendUint32(buf, m.nextSeg)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.segments))) //nolint:gosec // Segment ids are uint32.

	for _, s := range m.segments {
		buf = binary.LittleEndian.AppendUint32(buf, s.id)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(s.size)) //nolint:gosec // Sizes are non-negative.
		buf = binary.LittleEndian.AppendUint64(buf, uint64(s.dead)) //nolint:gosec // Sizes are non-negative.
	}

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.shards))) //nolint:gosec // At most maxShards.

	for _, sh := range m.shards {
		buf = binary.LittleEndian.AppendUint64(buf, sh.count)
		buf = binary.LittleEndian.AppendUint32(buf, sh.numPages)
		buf = binary.LittleEndian.AppendUint32(buf, uint32(sh.depth))

		for _, id := range sh.dir {
			buf = binary.LittleEndian.AppendUint32(buf, id)
		}
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

func (r *metaReader) u32() uint32 { return binary.LittleEndian.Uint32(r.take(u32Size)) }
func (r *metaReader) u64() uint64 { return binary.LittleEndian.Uint64(r.take(u64Size)) }

func decodeMeta(buf []byte) (*meta, error) {
	if len(buf) < len(metaMagic)+u32Size {
		return nil, fmt.Errorf("%w: meta file too short", ErrCorrupt)
	}

	body, sum := buf[:len(buf)-u32Size], binary.LittleEndian.Uint32(buf[len(buf)-u32Size:])
	if crc32.Checksum(body, castagnoli) != sum {
		return nil, fmt.Errorf("%w: meta checksum mismatch", ErrCorrupt)
	}

	reader := &metaReader{buf: body}

	if string(reader.take(len(metaMagic))) != metaMagic {
		return nil, fmt.Errorf("%w: not a diskmap directory", ErrCorrupt)
	}

	version := reader.u32()
	if version != metaVersion {
		return nil, fmt.Errorf("%w: unsupported format version %d", ErrCorrupt, version)
	}

	saved := &meta{seed: reader.u64(), nextSeg: reader.u32()}

	nsegs := reader.u32()
	if nsegs == 0 || int(nsegs) > len(reader.buf)/segmentMetaLen {
		return nil, fmt.Errorf("%w: bad segment count", ErrCorrupt)
	}

	for range nsegs {
		saved.segments = append(saved.segments, segmentMeta{
			id:   reader.u32(),
			size: int64(reader.u64()), //nolint:gosec // Written from non-negative int64.
			dead: int64(reader.u64()), //nolint:gosec // Written from non-negative int64.
		})
	}

	nshards := reader.u32()
	if nshards == 0 || nshards > maxShards || nshards&(nshards-1) != 0 {
		return nil, fmt.Errorf("%w: bad shard count %d", ErrCorrupt, nshards)
	}

	for range nshards {
		state, err := decodeShardMeta(reader)
		if err != nil {
			return nil, err
		}

		saved.shards = append(saved.shards, state)
	}

	if reader.err != nil || len(reader.buf) != 0 {
		return nil, fmt.Errorf("%w: bad meta length", ErrCorrupt)
	}

	return saved, nil
}

func decodeShardMeta(reader *metaReader) (shardMeta, error) {
	state := shardMeta{count: reader.u64(), numPages: reader.u32()}

	depth := reader.u32()
	if depth > maxDepth || u32Size<<depth > len(reader.buf) {
		return shardMeta{}, fmt.Errorf("%w: bad index depth", ErrCorrupt)
	}

	state.depth = uint8(depth)
	state.dir = make([]uint32, 1<<depth)

	for i := range state.dir {
		state.dir[i] = reader.u32()
		if state.dir[i] >= state.numPages {
			return shardMeta{}, fmt.Errorf("%w: directory points past the index", ErrCorrupt)
		}
	}

	return state, nil
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
	err := os.Remove(filepath.Join(dir, metaFile))
	if err != nil {
		return fmt.Errorf("diskmap: removing meta: %w", err)
	}

	return syncDir(dir)
}

// syncDir makes a rename or removal in dir durable. Windows cannot sync a
// directory and does not need to.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

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
		if name == metaFile || name == metaFile+".tmp" || isNumbered(name, indexSuffix) || isNumbered(name, segmentSuffix) {
			files = append(files, name)
		}
	}

	slices.Sort(files)

	return files, nil
}

// isNumbered reports whether name is a decimal number followed by suffix.
func isNumbered(name, suffix string) bool {
	stem, ok := strings.CutSuffix(name, suffix)
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
		err := os.Remove(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("diskmap: removing %s: %w", name, err)
		}
	}

	return nil
}
