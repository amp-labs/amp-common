package using

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testString = "test"

func TestUse_Success(t *testing.T) {
	t.Parallel()

	called := false
	closeCalled := false

	resource := NewResource(func() (string, Closer, error) {
		return testString + " value", func() error {
			closeCalled = true

			return nil
		}, nil
	})

	err := resource.Use(func(value string) error {
		assert.Equal(t, testString+" value", value)

		called = true

		return nil
	})

	require.NoError(t, err)
	assert.True(t, called, "function should have been called")
	assert.True(t, closeCalled, "closer should have been called")
}

func TestUse_NilResource(t *testing.T) {
	t.Parallel()

	var resource *Resource[string]

	err := resource.Use(func(value string) error {
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource is nil")
}

func TestUse_NilFunction(t *testing.T) {
	t.Parallel()

	resource := NewResource(func() (string, Closer, error) {
		return testString, nil, nil
	})

	err := resource.Use(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "f is nil")
}

func TestUse_ResourceReturnsError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("resource error") //nolint:err113

	resource := NewResource(func() (string, Closer, error) {
		return "", nil, expectedErr
	})

	err := resource.Use(func(value string) error {
		t.Fatal("should not be called")

		return nil
	})

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestUse_FunctionReturnsError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("function error") //nolint:err113
	closeCalled := false

	resource := NewResource(func() (string, Closer, error) {
		return testString, func() error {
			closeCalled = true

			return nil
		}, nil
	})

	err := resource.Use(func(value string) error {
		return expectedErr
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "function error")
	assert.True(t, closeCalled, "closer should still be called even when function errors")
}

func TestUse_CloserReturnsError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("closer error") //nolint:err113

	resource := NewResource(func() (string, Closer, error) {
		return testString, func() error {
			return expectedErr
		}, nil
	})

	err := resource.Use(func(value string) error {
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "closer error")
}

func TestUse_BothFunctionAndCloserReturnErrors(t *testing.T) {
	t.Parallel()

	funcErr := errors.New("function error") //nolint:err113
	closeErr := errors.New("closer error")  //nolint:err113

	resource := NewResource(func() (string, Closer, error) {
		return testString, func() error {
			return closeErr
		}, nil
	})

	err := resource.Use(func(value string) error {
		return funcErr
	})

	require.Error(t, err)
	// Should contain both errors
	assert.Contains(t, err.Error(), "function error")
	assert.Contains(t, err.Error(), "closer error")
}

func TestUse_NilCloser(t *testing.T) {
	t.Parallel()

	called := false

	resource := NewResource(func() (string, Closer, error) {
		return testString, nil, nil
	})

	err := resource.Use(func(value string) error {
		called = true

		return nil
	})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestCreateFile(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/test.txt"

	err := CreateFile(tmpFile).Use(func(f *os.File) error {
		_, err := f.WriteString("hello world")

		return err
	})

	require.NoError(t, err)

	// Verify file was created and closed
	content, err := os.ReadFile(tmpFile) //nolint:gosec // Safe in test
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

func TestOpenFile(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/test.txt"

	// Create file first
	err := os.WriteFile(tmpFile, []byte("test content"), 0o600)
	require.NoError(t, err)

	var content string

	err = OpenFile(tmpFile).Use(func(f *os.File) error {
		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		content = string(data)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, "test content", content)
}

func TestOpenFile_NonExistent(t *testing.T) {
	t.Parallel()

	err := OpenFile("/non/existent/file").Use(func(f *os.File) error {
		t.Fatal("should not be called")

		return nil
	})

	require.Error(t, err)
}

func TestFile(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/test.txt"

	f, err := os.Create(tmpFile) //nolint:gosec // Safe in test
	require.NoError(t, err)

	// File should be closed by the resource
	err = File(f).Use(func(f *os.File) error {
		_, err := f.WriteString("test")

		return err
	})

	require.NoError(t, err)

	// Verify file is closed
	_, err = f.WriteString("should fail")
	require.Error(t, err)
}

func TestWriter_WithWriteCloser(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/test.txt"

	f, err := os.Create(tmpFile) //nolint:gosec // Safe in test
	require.NoError(t, err)

	err = Writer(f).Use(func(w io.Writer) error {
		_, err := w.Write([]byte("test"))

		return err
	})

	require.NoError(t, err)

	// Verify file was closed
	_, err = f.WriteString("should fail")
	require.Error(t, err)

	// Verify content
	content, err := os.ReadFile(tmpFile) //nolint:gosec // Safe in test
	require.NoError(t, err)
	assert.Equal(t, "test", string(content))
}

func TestWriter_WithPlainWriter(t *testing.T) {
	t.Parallel()

	var buf strings.Builder

	err := Writer(&buf).Use(func(w io.Writer) error {
		_, err := w.Write([]byte("test"))

		return err
	})

	require.NoError(t, err)
	assert.Equal(t, "test", buf.String())
}

func TestReader_WithReadCloser(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/test.txt"

	err := os.WriteFile(tmpFile, []byte("test content"), 0o600)
	require.NoError(t, err)

	fileHandle, err := os.Open(tmpFile) //nolint:gosec // Safe in test
	require.NoError(t, err)

	var content string

	err = Reader(fileHandle).Use(func(r io.Reader) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		content = string(data)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, "test content", content)

	// Verify file was closed
	_, err = fileHandle.Read(make([]byte, 1))
	require.Error(t, err)
}

func TestReader_WithPlainReader(t *testing.T) {
	t.Parallel()

	r := strings.NewReader("test content")

	var content string

	err := Reader(r).Use(func(r io.Reader) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		content = string(data)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, "test content", content)
}

func TestReadWriter_WithReadWriteCloser(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/test.txt"

	f, err := os.Create(tmpFile) //nolint:gosec // Safe in test
	require.NoError(t, err)

	err = ReadWriter(f).Use(func(rw io.ReadWriter) error {
		_, err := rw.Write([]byte("test"))

		return err
	})

	require.NoError(t, err)

	// Verify file was closed
	_, err = f.WriteString("should fail")
	require.Error(t, err)
}

func TestReadWriter_WithPlainReadWriter(t *testing.T) {
	t.Parallel()

	var buf strings.Builder

	// strings.Builder only implements Writer, so we need a type that implements both
	type readWriter struct {
		io.Reader
		io.Writer
	}

	rw := &readWriter{
		Reader: strings.NewReader("input"),
		Writer: &buf,
	}

	err := ReadWriter(rw).Use(func(rw io.ReadWriter) error {
		_, err := rw.Write([]byte("test"))

		return err
	})

	require.NoError(t, err)
	assert.Equal(t, "test", buf.String())
}

func TestWrapCloser_WithNonNilCloser(t *testing.T) {
	t.Parallel()

	closed := false

	closer := &mockCloser{
		closeFunc: func() error {
			closed = true

			return nil
		},
	}

	wrapped := WrapCloser(closer)
	err := wrapped()

	require.NoError(t, err)
	assert.True(t, closed)
}

func TestWrapCloser_WithNilCloser(t *testing.T) {
	t.Parallel()

	wrapped := WrapCloser(nil)
	err := wrapped()

	assert.NoError(t, err)
}

func TestWrapCloser_WithErrorReturningCloser(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("close error") //nolint:err113

	closer := &mockCloser{
		closeFunc: func() error {
			return expectedErr
		},
	}

	wrapped := WrapCloser(closer)
	err := wrapped()

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// mockCloser is a helper type for testing.
type mockCloser struct {
	closeFunc func() error
}

func (m *mockCloser) Close() error {
	return m.closeFunc()
}
