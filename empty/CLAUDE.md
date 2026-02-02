# Package: empty

Create empty values of various types (slices, maps, structs).

## Usage

```go
// Empty collections
slice := empty.Slice[string]()  // []string{}
m := empty.Map[string, int]()   // map[string]int{}

// Empty struct for signaling
type T = empty.T
done := make(chan T)
done <- empty.V
```

## Common Patterns

- Distinguish between nil and empty collections
- Use `empty.T` and `empty.V` for struct{} channels
- Get pointers to empty collections with `SlicePtr()` and `MapPtr()`

## Gotchas

- Useful when APIs differentiate between nil and empty
- Not needed if nil is acceptable as "empty"
