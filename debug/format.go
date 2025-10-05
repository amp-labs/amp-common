package debug

import "encoding/json"

// PrettyJSONString returns the given value as a pretty-printed JSON string.
// If the value cannot be marshaled to JSON, an empty string is returned.
func PrettyJSONString(v any) string {
	//nolint:errchkjson
	jsonString, _ := json.MarshalIndent(v, "", "  ")

	return string(jsonString)
}
