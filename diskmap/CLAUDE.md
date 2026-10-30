# Package: diskmap

Disk-backed `map[string][]byte` for data that has outgrown memory, with a
configurable memory ceiling and predictable per-operation cost.

## Usage

```go
m, err := diskmap.Open("/var/lib/myapp/cache",
    diskmap.WithMemoryLimit(256<<20), // index cache + write buffers
    diskmap.WithValueCache(64<<20),   // optional, out of the same limit
)
if err != nil {
    return err
}
defer m.Close() // required: an unclosed map cannot be reopened

err = m.Add("user:42", payload)
value, found, err := m.Get("user:42")
buf, found, err = m.AppendValue(buf[:0], "user:42") // reuse a buffer
err = m.Remove("user:42")

err = m.Range(func(key string, value []byte) bool {
    return true // keep going
})
```

## Common Patterns

- `Open(dir, opts...)` creates the directory, or reopens a map closed with `Close`
- `WithDiscardExisting()` for scratch maps: wipe whatever is in the directory
- `AppendValue` instead of `Get` in hot loops to avoid an allocation per read
- `WithValueCache` when some keys are read far more than others
- `Stats()` for cache hit rates, disk usage and dead bytes awaiting compaction
- `Compact()` to reclaim all dead space now (normally automatic)

## Design

- Values go into append-only segment files; the index is an extendible hash
  table on disk, split into shards (`WithShards`, default 16), each with its
  own lock and a fixed-size page cache
- Every Get reads exactly one index page (usually cached) and one record
- File I/O happens outside all locks; dirty index pages are written back by a
  background goroutine per shard; a flusher writes double-buffered segment
  data; a compactor rewrites live records out of mostly-dead segments
- Memory held: about the memory limit, plus 8 bytes per index page and the
  directory (4 bytes per entry), roughly 0.1 bytes per key

## Gotchas

- Not durable: a crash leaves the directory unopenable (`ErrNotCleanlyClosed`);
  reopen with `WithDiscardExisting()` to start over
- One `Map` per directory at a time; a second `Open` fails as if crashed
- Disk use stays near live data / (1 - compaction threshold) plus one segment
- Range does not see entries added during iteration, and pauses compaction
- On copy-on-write filesystems (btrfs, ZFS) random index writes can stall for
  hundreds of milliseconds; disable CoW on the directory (`chattr +C`)
- Keys up to 64 KiB, values up to 1 GiB

## Related

- `maps` - in-memory map implementations
