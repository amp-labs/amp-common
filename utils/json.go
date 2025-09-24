package utils

import (
	"encoding/json"
	"fmt"
)

// ToJSONMap converts a struct to a maps with JSON-like keys.
func ToJSONMap(input any) (map[string]any, error) {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("error marshaling to JSON: %w", err)
	}

	var jsonMap map[string]any
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling from JSON: %w", err)
	}

	return jsonMap, nil
}
