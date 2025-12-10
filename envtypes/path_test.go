package envtypes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalPath_AsTuple(t *testing.T) {
	t.Parallel()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp(t.TempDir(), "envtypes_test_*.txt")
	require.NoError(t, err)

	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	// Get file info
	fileInfo, err := os.Stat(tmpFile.Name())
	require.NoError(t, err)

	localPath := LocalPath{
		Path: tmpFile.Name(),
		Info: fileInfo,
	}

	tuple := localPath.AsTuple()

	assert.Equal(t, localPath.Path, tuple.First())
	assert.Equal(t, localPath.Info, tuple.Second())
	assert.Equal(t, fileInfo.Name(), tuple.Second().Name())
	assert.Equal(t, fileInfo.Size(), tuple.Second().Size())
}

func TestLocalPath_WithDirectory(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Get directory info
	dirInfo, err := os.Stat(tmpDir)
	require.NoError(t, err)

	localPath := LocalPath{
		Path: tmpDir,
		Info: dirInfo,
	}

	tuple := localPath.AsTuple()

	assert.Equal(t, localPath.Path, tuple.First())
	assert.Equal(t, localPath.Info, tuple.Second())
	assert.True(t, tuple.Second().IsDir())
}

func TestLocalPath_FieldsAccessible(t *testing.T) {
	t.Parallel()

	// Create a temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "envtypes_field_test_*.txt")
	require.NoError(t, err)

	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	fileInfo, err := os.Stat(tmpFile.Name())
	require.NoError(t, err)

	localPath := LocalPath{
		Path: tmpFile.Name(),
		Info: fileInfo,
	}

	// Verify fields are directly accessible
	assert.NotEmpty(t, localPath.Path)
	assert.NotNil(t, localPath.Info)
	assert.Equal(t, tmpFile.Name(), localPath.Path)
	assert.False(t, localPath.Info.IsDir())
}
