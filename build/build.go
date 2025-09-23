package build

import (
	"encoding/json"
	"log/slog"
)

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
