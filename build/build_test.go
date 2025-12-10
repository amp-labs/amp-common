package build_test

import (
	"testing"

	"github.com/amp-labs/amp-common/build"
	"github.com/stretchr/testify/assert"
)

func TestParse_ValidJSON(t *testing.T) {
	t.Parallel()

	js := `{
		"git_commit": "abc123",
		"git_branch": "main",
		"git_date": "2025-10-05",
		"build_time": "2025-10-05T12:00:00Z",
		"build_host": "localhost",
		"build_user": "builder",
		"go_version": "go1.25.5",
		"dependencies": {
			"github.com/example/pkg": "v1.2.3"
		}
	}`

	info, ok := build.Parse(js)

	assert.True(t, ok)
	assert.NotNil(t, info)
	assert.Equal(t, "abc123", info.GitCommit)
	assert.Equal(t, "main", info.GitBranch)
	assert.Equal(t, "2025-10-05", info.GitDate)
	assert.Equal(t, "2025-10-05T12:00:00Z", info.BuildTime)
	assert.Equal(t, "localhost", info.BuildHost)
	assert.Equal(t, "builder", info.BuildUser)
	assert.Equal(t, "go1.25.5", info.GoVersion)
	assert.Equal(t, map[string]string{"github.com/example/pkg": "v1.2.3"}, info.Dependencies)
}

func TestParse_EmptyString(t *testing.T) {
	t.Parallel()

	info, ok := build.Parse("")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestParse_EmptyJSON(t *testing.T) {
	t.Parallel()

	info, ok := build.Parse("{}")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestParse_InvalidJSON(t *testing.T) {
	t.Parallel()

	info, ok := build.Parse("not valid json")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestParse_PartialJSON(t *testing.T) {
	t.Parallel()

	js := `{
		"git_commit": "abc123",
		"git_branch": "main"
	}`

	info, ok := build.Parse(js)

	assert.True(t, ok)
	assert.NotNil(t, info)
	assert.Equal(t, "abc123", info.GitCommit)
	assert.Equal(t, "main", info.GitBranch)
	assert.Empty(t, info.GitDate)
	assert.Empty(t, info.BuildTime)
	assert.Empty(t, info.BuildHost)
	assert.Empty(t, info.BuildUser)
	assert.Empty(t, info.GoVersion)
	assert.Nil(t, info.Dependencies)
}

func TestParse_EmptyDependencies(t *testing.T) {
	t.Parallel()

	js := `{
		"git_commit": "abc123",
		"dependencies": {}
	}`

	info, ok := build.Parse(js)

	assert.True(t, ok)
	assert.NotNil(t, info)
	assert.Equal(t, "abc123", info.GitCommit)
	assert.NotNil(t, info.Dependencies)
	assert.Empty(t, info.Dependencies)
}

func TestParse_MultipleDependencies(t *testing.T) {
	t.Parallel()

	js := `{
		"dependencies": {
			"github.com/example/pkg1": "v1.0.0",
			"github.com/example/pkg2": "v2.0.0",
			"github.com/example/pkg3": "v3.0.0"
		}
	}`

	info, ok := build.Parse(js)

	assert.True(t, ok)
	assert.NotNil(t, info)
	assert.Len(t, info.Dependencies, 3)
	assert.Equal(t, "v1.0.0", info.Dependencies["github.com/example/pkg1"])
	assert.Equal(t, "v2.0.0", info.Dependencies["github.com/example/pkg2"])
	assert.Equal(t, "v3.0.0", info.Dependencies["github.com/example/pkg3"])
}
