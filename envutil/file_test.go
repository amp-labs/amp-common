package envutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvFile(t *testing.T) {
	t.Parallel()

	t.Run(".env file - valid key-value pairs", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])
		assert.Equal(t, "myapp", vars["DB_NAME"])
		assert.Len(t, vars, 3)
	})

	t.Run(".env file - with comments", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `# Database configuration
DB_HOST=localhost
# Port number
DB_PORT=5432`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])
		assert.Len(t, vars, 2)
	})

	t.Run(".env file - with empty lines", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `DB_HOST=localhost

DB_PORT=5432


DB_NAME=myapp
`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])
		assert.Equal(t, "myapp", vars["DB_NAME"])
		assert.Len(t, vars, 3)
	})

	t.Run(".env file - values with equals signs", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `SECRET_KEY=abc=def=ghi
CONNECTION_STRING=postgresql://user:pass=word@host:5432/db`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "abc=def=ghi", vars["SECRET_KEY"])
		assert.Equal(t, "postgresql://user:pass=word@host:5432/db", vars["CONNECTION_STRING"])
	})

	t.Run(".env file - whitespace trimming", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `  DB_HOST  =  localhost
DB_PORT=  5432
  DB_NAME=myapp  `)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])
		assert.Equal(t, "myapp", vars["DB_NAME"])
	})

	t.Run(".env file - lines without equals sign are ignored", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `DB_HOST=localhost
INVALID_LINE_WITHOUT_EQUALS
DB_PORT=5432`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])
		assert.Len(t, vars, 2)
	})

	t.Run(".env file - empty file", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", "")

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Empty(t, vars)
	})

	t.Run(".env file - only comments", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `# Comment 1
# Comment 2
# Comment 3`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Empty(t, vars)
	})

	t.Run(".json file - valid structure", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.json", `{
  "env": {
    "DB_HOST": "localhost",
    "DB_PORT": "5432",
    "DB_NAME": "myapp"
  }
}`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])
		assert.Equal(t, "myapp", vars["DB_NAME"])
		assert.Len(t, vars, 3)
	})

	t.Run(".json file - empty env object", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.json", `{"env": {}}`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Empty(t, vars)
	})

	t.Run(".json file - no env field", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.json", `{"other": "data"}`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Nil(t, vars)
	})

	t.Run(".json file - invalid JSON", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.json", `{invalid json`)

		_, err := envutil.LoadEnvFile(tmpfile)
		require.Error(t, err)
	})

	t.Run(".yml file - valid structure", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.yml", `env:
  DB_HOST: localhost
  DB_PORT: "5432"
  DB_NAME: myapp`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])
		assert.Equal(t, "myapp", vars["DB_NAME"])
		assert.Len(t, vars, 3)
	})

	t.Run(".yaml file - valid structure", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.yaml", `env:
  DB_HOST: localhost
  DB_PORT: "5432"`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
		assert.Equal(t, "5432", vars["DB_PORT"])
		assert.Len(t, vars, 2)
	})

	t.Run(".yaml file - empty env object", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.yaml", `env: {}`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Empty(t, vars)
	})

	t.Run(".yaml file - no env field", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.yaml", `other: data`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Nil(t, vars)
	})

	t.Run(".yaml file - invalid YAML", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.yaml", `env:
  - this is
  - invalid: structure
  - for env vars`)

		_, err := envutil.LoadEnvFile(tmpfile)
		require.Error(t, err)
	})

	t.Run("case-insensitive extension matching - .ENV", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.ENV", `DB_HOST=localhost`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "localhost", vars["DB_HOST"])
	})

	t.Run("case-insensitive extension matching - .JSON", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.JSON", `{"env": {"KEY": "value"}}`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "value", vars["KEY"])
	})

	t.Run("case-insensitive extension matching - .YML", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.YML", `env:
  KEY: value`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "value", vars["KEY"])
	})

	t.Run("case-insensitive extension matching - .YAML", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.YAML", `env:
  KEY: value`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "value", vars["KEY"])
	})

	t.Run("unknown file extension", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.txt", `DB_HOST=localhost`)

		_, err := envutil.LoadEnvFile(tmpfile)
		require.Error(t, err)
		assert.ErrorIs(t, err, envutil.ErrUnknownFileType)
	})

	t.Run("file does not exist", func(t *testing.T) {
		t.Parallel()

		_, err := envutil.LoadEnvFile("/nonexistent/path/to/file.env")
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("edge case - empty key", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `=value
KEY=value`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		// Empty key should be included
		assert.Equal(t, "value", vars[""])
		assert.Equal(t, "value", vars["KEY"])
		assert.Len(t, vars, 2)
	})

	t.Run("edge case - empty value", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "test.env", `EMPTY_KEY=
KEY=value`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Empty(t, vars["EMPTY_KEY"])
		assert.Equal(t, "value", vars["KEY"])
	})

	t.Run("multiple extensions in filename", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "config.production.env", `KEY=value`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Equal(t, "value", vars["KEY"])
	})

	t.Run("realistic .env example", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, ".env", `# Application configuration
APP_NAME=MyApp
APP_ENV=production

# Database configuration
DATABASE_URL=postgresql://user:password@localhost:5432/mydb
DB_POOL_SIZE=10

# API Keys (can contain special characters)
API_KEY=sk_test_abc123=def456
SECRET=my-secret-key-with-dashes

# Feature flags
FEATURE_X_ENABLED=true
FEATURE_Y_ENABLED=false`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Len(t, vars, 8)
		assert.Equal(t, "MyApp", vars["APP_NAME"])
		assert.Equal(t, "production", vars["APP_ENV"])
		assert.Equal(t, "postgresql://user:password@localhost:5432/mydb", vars["DATABASE_URL"])
		assert.Equal(t, "sk_test_abc123=def456", vars["API_KEY"])
		assert.Equal(t, "true", vars["FEATURE_X_ENABLED"])
	})

	t.Run("realistic .json example", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "config.json", `{
  "version": "1.0",
  "env": {
    "APP_NAME": "MyApp",
    "APP_ENV": "production",
    "DATABASE_URL": "postgresql://user:password@localhost:5432/mydb",
    "API_KEY": "sk_test_abc123"
  },
  "metadata": {
    "created": "2024-01-01"
  }
}`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Len(t, vars, 4)
		assert.Equal(t, "MyApp", vars["APP_NAME"])
		assert.Equal(t, "production", vars["APP_ENV"])
		assert.Equal(t, "postgresql://user:password@localhost:5432/mydb", vars["DATABASE_URL"])
	})

	t.Run("realistic .yaml example", func(t *testing.T) {
		t.Parallel()

		tmpfile := createTempFile(t, "config.yaml", `version: "1.0"
env:
  APP_NAME: MyApp
  APP_ENV: production
  DATABASE_URL: postgresql://user:password@localhost:5432/mydb
  API_KEY: sk_test_abc123
metadata:
  created: "2024-01-01"`)

		vars, err := envutil.LoadEnvFile(tmpfile)
		require.NoError(t, err)
		assert.Len(t, vars, 4)
		assert.Equal(t, "MyApp", vars["APP_NAME"])
		assert.Equal(t, "production", vars["APP_ENV"])
	})
}

// createTempFile creates a temporary file with the given name and content.
// The file is automatically cleaned up when the test completes.
func createTempFile(t *testing.T, filename, content string) string {
	t.Helper()

	tmpdir := t.TempDir()
	tmpfile := filepath.Join(tmpdir, filename)

	err := os.WriteFile(tmpfile, []byte(content), 0600)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Remove(tmpfile)
		if err != nil {
			t.Logf("Failed to remove temporary file %s: %s", tmpfile, err)
		}
	})

	return tmpfile
}
