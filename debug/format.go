package debug

import "encoding/json"

//nolint:errchkjson
func PrettyJSONString(v any) string {
	jsonString, _ := json.MarshalIndent(v, "", "  ")

	return string(jsonString)
}
