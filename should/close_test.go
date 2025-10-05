package should_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/amp-labs/amp-common/should"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errCloseFailed    = errors.New("close failed")
	errError1         = errors.New("error 1")
	errError2         = errors.New("error 2")
	errBenchmarkError = errors.New("benchmark error")
)

type mockCloser struct {
	closeErr error
	closed   bool
}

func (m *mockCloser) Close() error {
	m.closed = true

	return m.closeErr
}

func TestClose_Success(t *testing.T) {
	t.Parallel()

	closer := &mockCloser{}

	should.Close(closer, "test message")

	assert.True(t, closer.closed, "Close should have been called")
}

func TestClose_Failure(t *testing.T) {
	t.Parallel()

	closer := &mockCloser{closeErr: errCloseFailed}

	should.Close(closer, "failed to close resource")

	assert.True(t, closer.closed, "Close should have been called")
}

func TestClose_NilCloser(t *testing.T) {
	t.Parallel()

	// This will panic, which is expected behavior for nil closers
	assert.Panics(t, func() {
		should.Close(nil, "test message")
	}, "Calling Close on nil should panic")
}

func TestClose_RealFile(t *testing.T) {
	t.Parallel()

	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	file, err := os.Create(tmpFile)
	require.NoError(t, err)

	// Write some data
	_, err = file.WriteString("test data")
	require.NoError(t, err)

	// Close the file
	should.Close(file, "failed to close file")

	// Verify file is closed by trying to write (should fail)
	_, err = file.WriteString("more data")
	assert.Error(t, err, "Writing to closed file should fail")
}

func TestClose_AlreadyClosedFile(t *testing.T) {
	t.Parallel()

	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	file, err := os.Create(tmpFile)
	require.NoError(t, err)

	// Close it once
	err = file.Close()
	require.NoError(t, err)

	// Try to close again - this should log an error but not panic
	should.Close(file, "failed to close already-closed file")
}

func TestRemove_Success(t *testing.T) {
	t.Parallel()

	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0o600)
	require.NoError(t, err)

	should.Remove(tmpFile, "failed to remove file")

	// Verify file is removed
	_, err = os.Stat(tmpFile)
	assert.True(t, os.IsNotExist(err), "File should be removed")
}

func TestRemove_NonExistentFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist.txt")

	// This should log an error but not panic
	should.Remove(nonExistent, "failed to remove non-existent file")
}

func TestRemove_Directory(t *testing.T) {
	t.Parallel()

	// Create a temporary directory with a file inside
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0o755)
	require.NoError(t, err)

	// Create a file inside the directory
	tmpFile := filepath.Join(subDir, "test.txt")
	err = os.WriteFile(tmpFile, []byte("test"), 0o600)
	require.NoError(t, err)

	// Try to remove the directory (will fail because it's not empty)
	// This should log an error but not panic
	should.Remove(subDir, "failed to remove directory")

	// Verify directory still exists
	_, err = os.Stat(subDir)
	assert.NoError(t, err, "Directory should still exist")
}

func TestRemove_EmptyDirectory(t *testing.T) {
	t.Parallel()

	// Create an empty temporary directory
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	err := os.Mkdir(emptyDir, 0o755)
	require.NoError(t, err)

	should.Remove(emptyDir, "failed to remove empty directory")

	// Verify directory is removed
	_, err = os.Stat(emptyDir)
	assert.True(t, os.IsNotExist(err), "Directory should be removed")
}

func TestRemove_InDefer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	func() {
		tmpFile := filepath.Join(tmpDir, "defer-test.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0o600)
		require.NoError(t, err)

		defer should.Remove(tmpFile, "failed to remove in defer")

		// Do some work...
		data, err := os.ReadFile(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, "test", string(data))
	}()

	// File should be removed after function returns
	_, err := os.Stat(filepath.Join(tmpDir, "defer-test.txt"))
	assert.True(t, os.IsNotExist(err), "File should be removed by defer")
}

func TestClose_InDefer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	var file *os.File

	func() {
		var err error

		tmpFile := filepath.Join(tmpDir, "defer-test.txt")
		file, err = os.Create(tmpFile)
		require.NoError(t, err)

		defer should.Close(file, "failed to close in defer")

		_, err = file.WriteString("test data")
		require.NoError(t, err)
	}()

	// Verify file is closed
	_, err := file.WriteString("more data")
	assert.Error(t, err, "File should be closed by defer")
}

// multiple times in a defer chain.
func TestClose_WithMultipleErrors(t *testing.T) {
	t.Parallel()

	closer1 := &mockCloser{closeErr: errError1}
	closer2 := &mockCloser{closeErr: errError2}
	closer3 := &mockCloser{}

	func() {
		defer should.Close(closer1, "closer1 failed")
		defer should.Close(closer2, "closer2 failed")
		defer should.Close(closer3, "closer3 failed")
	}()

	assert.True(t, closer1.closed)
	assert.True(t, closer2.closed)
	assert.True(t, closer3.closed)
}

// Benchmark tests.
func BenchmarkClose_Success(b *testing.B) {
	closer := &mockCloser{}

	b.ResetTimer()

	for range b.N {
		closer.closed = false
		should.Close(closer, "benchmark message")
	}
}

func BenchmarkClose_WithError(b *testing.B) {
	closer := &mockCloser{closeErr: errBenchmarkError}

	b.ResetTimer()

	for range b.N {
		closer.closed = false
		should.Close(closer, "benchmark message")
	}
}

func BenchmarkRemove_Success(b *testing.B) {
	tmpDir := b.TempDir()

	for range b.N {
		tmpFile := filepath.Join(tmpDir, "bench.txt")
		_ = os.WriteFile(tmpFile, []byte("test"), 0o600)

		should.Remove(tmpFile, "benchmark message")
	}
}

// Example of io.Closer that implements additional cleanup.
type complexCloser struct {
	resources []io.Closer
}

func (c *complexCloser) Close() error {
	for _, r := range c.resources {
		if err := r.Close(); err != nil {
			return err
		}
	}

	return nil
}

func TestClose_ComplexCloser(t *testing.T) {
	t.Parallel()

	mock1 := &mockCloser{}
	mock2 := &mockCloser{}
	multiCloser := &complexCloser{
		resources: []io.Closer{mock1, mock2},
	}

	should.Close(multiCloser, "failed to close complex resource")

	assert.True(t, mock1.closed)
	assert.True(t, mock2.closed)
}
