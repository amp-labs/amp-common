# Package: compare

Interface for types that can compare themselves for equality.

## Usage

```go
type MyType struct { Value int }

func (m MyType) Equals(other MyType) bool {
    return m.Value == other.Value
}

// Use with compare utilities
result := compare.Equals(myObj, otherObj)
```

## Common Patterns

- Implement `Comparable[T]` for custom equality logic
- Used as building block for `sortable` and `collectable` interfaces
- Alternative to using == operator for complex types

## Gotchas

- Your `Equals()` method defines semantic equality
- Must be consistent (transitive, reflexive, symmetric)
