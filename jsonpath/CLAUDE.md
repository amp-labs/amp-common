# Package: jsonpath

JSONPath bracket notation utilities for field mapping.

## Usage

```go
// Parse JSONPath
segments, err := jsonpath.ParsePath("$['address']['city']")
// Returns: [{Key: "address"}, {Key: "city"}]

// Get value from map
data := map[string]any{"address": map[string]any{"city": "NYC"}}
value, err := jsonpath.GetValue(data, "$['address']['city']")
// Returns: "NYC"

// Case-insensitive matching
value, err := jsonpath.GetValueCaseInsensitive(data, "$['ADDRESS']['CITY']")
```

## Common Patterns

- Bracket notation: `$['field']['nestedField']`
- `ParsePath()` - Parse into segments
- `GetValue()` - Navigate nested maps (case-sensitive)
- `GetValueCaseInsensitive()` - Case-insensitive navigation
- `SetValue()` - Set values in nested maps

## Gotchas

- Only supports bracket notation (not dot notation)
- Path must start with `$[`
- Empty segments not allowed
- Returns specific errors for each failure type
