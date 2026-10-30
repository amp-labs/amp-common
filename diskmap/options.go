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
	hash                func(key string, seed uint64) uint64
}

func defaultOptions() options {
	return options{
		memoryLimit:         DefaultMemoryLimit,
		segmentSize:         DefaultSegmentSize,
		compactionThreshold: DefaultCompactionThreshold,
		verifyChecksums:     true,
	}
}

func (o *options) validate() error {
	if o.memoryLimit < minMemoryLimit {
		return fmt.Errorf("%w: memory limit must be at least %d bytes", ErrInvalidOption, minMemoryLimit)
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

// WithMemoryLimit sets the in-memory budget, in bytes, for the index page
// cache and the write buffer. The map also keeps a small directory that grows
// by four bytes per index page (about 0.03 bytes per key) and a few bytes per
// segment; those are not counted against the limit. Defaults to
// DefaultMemoryLimit; the minimum is 1 MiB.
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
