package envutil

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// maxSplitParts is the maximum number of parts to split on when parsing KEY=VALUE.
	// We use 2 to split on the first equals sign only, allowing values to contain '='.
	maxSplitParts = 2
)

// ErrUnknownFileType is returned when the file extension is not recognized.
var ErrUnknownFileType = errors.New("env file doesn't have a known file suffix")

// LoadEnvFile loads environment variables from a file and returns them as a map.
// The file format is automatically detected based on the file extension:
//   - .env files are parsed as key=value pairs (one per line)
//   - .json files are expected to have an "env" field containing string key-value pairs
//   - .yml/.yaml files are expected to have an "env" field containing string key-value pairs
//
// Example usage:
//
//	vars, err := LoadEnvFile("/path/to/.env")
//	if err != nil {
//	    return err
//	}
//	for key, value := range vars {
//	    os.Setenv(key, value)
//	}
//
// Returns an error if:
//   - The file doesn't exist or can't be read
//   - The file extension is not recognized (.env, .json, .yml, .yaml)
//   - The file content cannot be parsed according to its format
func LoadEnvFile(path string) (map[string]string, error) {
	// Check if the file exists and get its name for extension detection
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Use lowercase file name for case-insensitive extension matching
	name := strings.ToLower(fileInfo.Name())

	// Route to appropriate parser based on file extension
	switch {
	case strings.HasSuffix(name, ".env"):
		return loadEnvFile(path)
	case strings.HasSuffix(name, ".json"):
		return loadJSONFile(path)
	case strings.HasSuffix(name, ".yml"), strings.HasSuffix(name, ".yaml"):
		return loadYAMLFile(path)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownFileType, fileInfo.Name())
	}
}

// loadEnvFile parses a .env file and returns environment variables as a map.
// The .env file format supports:
//   - Key-value pairs in the format KEY=VALUE (one per line)
//   - Comments starting with # (entire line is ignored)
//   - Empty lines (ignored)
//   - Whitespace around keys and values is trimmed
//
// Example .env file:
//
//	# Database configuration
//	DB_HOST=localhost
//	DB_PORT=5432
//	DB_NAME=myapp
//
// Lines that don't contain an equals sign are silently ignored.
func loadEnvFile(path string) (map[string]string, error) {
	bts, err := os.ReadFile(path) // #nosec G304 -- path is the intended file to load
	if err != nil {
		return nil, err
	}

	// Use scanner for efficient line-by-line processing
	scanner := bufio.NewScanner(bytes.NewReader(bts))

	out := make(map[string]string)

	// Process each line in the file
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comment lines starting with #
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Split on the first equals sign to allow values to contain '='
		// For example: "SECRET_KEY=abc=def" becomes ["SECRET_KEY", "abc=def"]
		parts := strings.SplitN(line, "=", maxSplitParts)
		if len(parts) != maxSplitParts {
			// Skip lines without an equals sign (empty lines, malformed lines)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		out[key] = value
	}

	// Check if scanner encountered any errors during reading
	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return out, nil
}

// jsonEnvFile represents the expected structure of a JSON environment file.
// The JSON file must have an "env" field containing a map of string key-value pairs.
//
// Example JSON file:
//
//	{
//	  "env": {
//	    "DB_HOST": "localhost",
//	    "DB_PORT": "5432"
//	  }
//	}
type jsonEnvFile struct {
	Env map[string]string `json:"env"`
}

// loadJSONFile parses a JSON file and extracts environment variables from the "env" field.
// The JSON file must contain a top-level "env" object with string key-value pairs.
// Returns an error if the file cannot be read or parsed as valid JSON.
func loadJSONFile(path string) (map[string]string, error) {
	bts, err := os.ReadFile(path) // #nosec G304 -- path is the intended file to load
	if err != nil {
		return nil, err
	}

	out := &jsonEnvFile{}

	// Parse JSON content into the jsonEnvFile struct
	err = json.Unmarshal(bts, &out)
	if err != nil {
		return nil, err
	}

	// Extract and return the env map
	return out.Env, nil
}

// yamlEnvFile represents the expected structure of a YAML environment file.
// The YAML file must have an "env" field containing a map of string key-value pairs.
//
// Example YAML file:
//
//	env:
//	  DB_HOST: localhost
//	  DB_PORT: "5432"
//	  DB_NAME: myapp
type yamlEnvFile struct {
	Env map[string]string `yaml:"env"`
}

// loadYAMLFile parses a YAML file and extracts environment variables from the "env" field.
// The YAML file must contain a top-level "env" object with string key-value pairs.
// Returns an error if the file cannot be read or parsed as valid YAML.
func loadYAMLFile(path string) (map[string]string, error) {
	bts, err := os.ReadFile(path) // #nosec G304 -- path is the intended file to load
	if err != nil {
		return nil, err
	}

	env := &yamlEnvFile{}

	// Parse YAML content into the yamlEnvFile struct
	err = yaml.Unmarshal(bts, &env)
	if err != nil {
		return nil, err
	}

	// Extract and return the env map
	return env.Env, nil
}
