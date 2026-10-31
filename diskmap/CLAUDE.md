# Package: diskmap

Disk-backed `map[string][]byte` for data that has outgrown memory, with a
configurable memory ceiling, predictable per-operation cost, and crash
recovery.

## Usage

```go
m, err := diskmap.Open("/var/lib/myapp/cache",
    diskmap.WithMemoryLimit(256<<20), // index cache + write buffers
    diskmap.WithValueCache(64<<20),   // optional, out of the same limit
)
if err != nil {
    return err
}
defer m.Close()

err = m.Add("user:42", payload)       // survives a crash once it returns
value, found, err := m.Get("user:42")
buf, found, err = m.AppendValue(buf[:0], "user:42") // reuse a buffer
err = m.Remove("user:42")
err = m.Sync()                        // survive a power failure too

err = m.Range(func(key string, value []byte) bool {
    return true // keep going
})
```

## Common Patterns

- `Open(dir, opts...)` creates the directory, reopens a closed map, or
  recovers one whose process crashed
- Durability: default = each write reaches the OS before returning (crash-safe);
  `Sync()` fsyncs; `WithSyncWrites()` fsyncs every write (group commit);
  `WithBufferedWrites()` for bulk loads (fast, may lose the last few ms)
- `WithDiscardExisting()` for scratch maps: wipe whatever is in the directory
- `AppendValue` instead of `Get` in hot loops to avoid an allocation per read
- `WithValueCache` when some keys are read far more than others
- `Stats()` for cache hit rates, disk usage, checkpoints and write stalls

## Design

- Values go into append-only segment files; every put and tombstone has a
  sequence number, so the segments are a replayable log
- The index is an extendible hash table on disk, split into shards
  (`WithShards`, default 16), each with its own lock and fixed-size page cache
- Index pages are copy-on-write through a logical-to-physical page table;
  checkpoints (every `WithCheckpointInterval` bytes, default 128 MiB) snapshot
  it after a brief freeze that copies each shard's directory and page table,
  and recovery replays only the log after the last checkpoint, stopping at
  the first gap (an exact prefix)
- File I/O happens outside all locks; writers share write syscalls (and
  fsyncs) through leader-based group commit
- Compaction copies live records into its own segments as "move" records and
  deletes old segments only after the next checkpoint is durable

## Gotchas

- One `Map` per directory; a second `Open` returns `ErrLocked` (not detected
  on Windows)
- Crash-safe writes cost a write syscall each (concurrent writers share it);
  `WithSyncWrites` costs an fsync per write, which is milliseconds on many disks
- Disk use stays near live data / (1 - compaction threshold) plus a few segments
- Range does not see entries added during iteration, and pauses compaction
- On copy-on-write filesystems (btrfs, ZFS) random index writes can stall;
  disable CoW on the directory (`chattr +C`)
- Keys up to 64 KiB, values up to 1 GiB - 1

## Related

- `maps` - in-memory map implementations
