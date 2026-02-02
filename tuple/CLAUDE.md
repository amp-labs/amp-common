# Package: tuple

Generic tuple types (Tuple2, Tuple3, Tuple4, Tuple5) for grouping multiple values together.

## Usage

```go
// Create a pair
pair := tuple.NewTuple2("name", 42)
name := pair.First()   // "name"
age := pair.Second()   // 42

// Create a triple
triple := tuple.NewTuple3("x", 1.5, true)
```

## Common Patterns

- Return multiple values from functions as a single type
- Use as map keys when you need composite keys
- Tuples are immutable with type-safe accessors

## Gotchas

- Tuples are immutable - create new tuples to change values
- Supports up to 5 elements (Tuple2 through Tuple5)
