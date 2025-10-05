// Package build provides utilities for parsing build information that is
// embedded at compile time. The Info struct is designed to be populated via
// -ldflags during the build process in other repositories.
package build

import (
	"encoding/json"
	"log/slog"
)

// Info contains build metadata that is typically embedded at compile time.
// This struct is populated in other repositories using -ldflags to inject
// version information, build details, and dependency versions.
type Info struct {
	GitCommit    string            `json:"git_commit"` //nolint:tagliatelle
	GitBranch    string            `json:"git_branch"` //nolint:tagliatelle
	GitDate      string            `json:"git_date"`   //nolint:tagliatelle
	BuildTime    string            `json:"build_time"` //nolint:tagliatelle
	BuildHost    string            `json:"build_host"` //nolint:tagliatelle
	BuildUser    string            `json:"build_user"` //nolint:tagliatelle
	GoVersion    string            `json:"go_version"` //nolint:tagliatelle
	Dependencies map[string]string `json:"dependencies"`
}

// Parse deserializes a JSON string into build Info.
// Returns (nil, false) if the input is empty, "{}", or fails to parse.
func Parse(js string) (*Info, bool) {
	if len(js) == 0 {
		return nil, false
	}

	if js == "{}" {
		return nil, false
	}

	var info Info

	err := json.Unmarshal([]byte(js), &info)
	if err != nil {
		slog.Warn("Failed to parse build info from JSON",
			"data", js,
			"error", err)

		return nil, false
	}

	return &info, true
}
