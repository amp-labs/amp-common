# Package: set

Generic set implementations with hash-based and red-black tree variants, plus thread-safe wrappers.

## Usage

```go
// Hash-based set (O(1) operations)
s := set.NewSet[MyType](hashing.Sha256)
s.Add(item)
contains, err := s.Contains(item)

// Red-black tree set (sorted, O(log n))
sortedSet := set.NewRedBlackTreeSet[sortable.Int]()
sortedSet.Add(sortable.Int(42))

// Thread-safe wrapper
safeSet := set.NewThreadSafeSet(s)
```

## Common Patterns

- `NewSet()` - Hash-based set for `Collectable` types
- `NewRedBlackTreeSet()` - Sorted set for `Sortable` types
- `NewThreadSafeSet()` - Thread-safe wrapper
- Operations: Add, Remove, Contains, Union, Intersection, Filter
- Ordered variants maintain insertion order

## Gotchas

- Hash-based sets require `Collectable` types
- Sorted sets require `Sortable` types
- Hash collisions return errors
- Thread-safe wrappers add mutex overhead

## Related

- `collectable`, `sortable` - Element types
- `maps` - Map implementations
