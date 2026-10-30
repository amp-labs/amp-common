package diskmap

import (
	"fmt"
)

const (
	// DefaultMemoryLimit is the default in-memory budget for caches and buffers.
	DefaultMemoryLimit = 64 << 20

	// DefaultSegmentSize is the default size at which a segment file is sealed
	// and a new one started.
	DefaultSegmentSize = 256 << 20

	// DefaultCompactionThreshold is the default fraction of dead bytes on disk
	// at which compaction starts reclaiming space.
	DefaultCompactionThreshold = 0.5

	// DefaultShards is the default number of index shards.
	DefaultShards = 16

	maxShards      = 256
	minMemoryLimit = 1 << 20
	minSegmentSize = 1 << 20
	maxSegmentSize = 2 << 30
)

type options struct {
	memoryLimit         int
	segmentSize         int64
	compactionThreshold float64
	verifyChecksums     bool
	discardExisting     bool
	shards              int
	valueCache          int
	hash                func(key string, seed uint64) uint64
}

func defaultOptions() options {
	return options{
		memoryLimit:         DefaultMemoryLimit,
		segmentSize:         DefaultSegmentSize,
		compactionThreshold: DefaultCompactionThreshold,
		verifyChecksums:     true,
		shards:              DefaultShards,
	}
}

func (o *options) validate() error {
	if o.shards < 1 || o.shards > maxShards || o.shards&(o.shards-1) != 0 {
		return fmt.Errorf("%w: shards must be a power of two between 1 and %d", ErrInvalidOption, maxShards)
	}

	if o.valueCache < 0 || o.valueCache > o.memoryLimit/2 {
		return fmt.Errorf("%w: value cache must be between 0 and half the memory limit", ErrInvalidOption)
	}

	least := max(minMemoryLimit, o.valueCache+2*minWriteBuffer+chunkSize+o.shards*(minFrames+minWriteBack)*pageSize)
	if o.memoryLimit < least {
		return fmt.Errorf("%w: memory limit must be at least %d bytes", ErrInvalidOption, least)
	}

	if o.segmentSize < minSegmentSize || o.segmentSize > maxSegmentSize {
		return fmt.Errorf("%w: segment size must be between %d and %d bytes",
			ErrInvalidOption, minSegmentSize, maxSegmentSize)
	}

	if o.compactionThreshold < 0 || o.compactionThreshold >= 1 {
		return fmt.Errorf("%w: compaction threshold must be in [0, 1)", ErrInvalidOption)
	}

	return nil
}

// Option configures a Map in Open.
type Option func(*options)

// WithMemoryLimit sets the in-memory budget, in bytes: the index page caches,
// the write buffers, the compactor's read buffer and any value cache all come
// out of it. Bookkeeping that grows with the map is not counted: about 12
// bytes per index page (roughly 0.1 bytes per key) and a few bytes per
// segment. Each Range in progress also uses a 64 KiB buffer. Defaults to
// DefaultMemoryLimit; the minimum is 1 MiB, more with many shards.
func WithMemoryLimit(bytes int) Option {
	return func(o *options) {
		o.memoryLimit = bytes
	}
}

// WithSegmentSize sets the size at which a segment file is sealed and a new
// one started. Smaller segments let compaction reclaim space in smaller steps;
// larger segments mean fewer open files. Defaults to DefaultSegmentSize;
// allowed range is 1 MiB to 2 GiB.
func WithSegmentSize(bytes int64) Option {
	return func(o *options) {
		o.segmentSize = bytes
	}
}

// WithCompactionThreshold sets the fraction of dead bytes (from overwritten or
// removed entries) at which the map starts compacting segments in small steps
// alongside writes. Disk usage stays near live/(1-threshold). Zero disables
// automatic compaction; Compact still works. Defaults to
// DefaultCompactionThreshold.
func WithCompactionThreshold(ratio float64) Option {
	return func(o *options) {
		o.compactionThreshold = ratio
	}
}

// WithShards sets how many independently locked shards the index is split
// into. More shards let more goroutines use the map at once; each costs an
// index file and a slice of the memory limit. It must be a power of two up to
// 256 and only applies when a map is created; a reopened map keeps its
// original count. Defaults to DefaultShards.
func WithShards(n int) Option {
	return func(o *options) {
		o.shards = n
	}
}

// WithValueCache sets aside bytes of the memory limit to keep recently read
// values in memory, so repeated reads of the same keys skip the file read.
// It pays off when some keys are read much more often than others; for
// uniformly random reads the memory does more good caching the index, which
// is where it goes by default. At most half the memory limit. Defaults to 0.
func WithValueCache(bytes int) Option {
	return func(o *options) {
		o.valueCache = bytes
	}
}

// WithVerifyChecksums controls whether reads verify each record's CRC-32C.
// Verification is on by default and costs a small amount of CPU per byte read.
func WithVerifyChecksums(enabled bool) Option {
	return func(o *options) {
		o.verifyChecksums = enabled
	}
}

// WithDiscardExisting deletes any map already stored in the directory,
// including one that was not closed cleanly, and starts empty. Only the map's
// own files are removed.
func WithDiscardExisting() Option {
	return func(o *options) {
		o.discardExisting = true
	}
}

// withHash replaces the key hash. Tests use it to force collisions.
func withHash(hash func(key string, seed uint64) uint64) Option {
	return func(o *options) {
		o.hash = hash
	}
}
