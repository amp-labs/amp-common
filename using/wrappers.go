package using

import (
	"fmt"
	"io"
	"os"
)

// CreateFile returns a Resource that creates a new file at the given path.
// The file will be created with mode 0666 (before umask) and truncated if it already exists.
// The file is automatically closed when the Resource is used.
func CreateFile(path string) *Resource[*os.File] {
	return NewResource(func() (*os.File, Closer, error) {
		f, err := os.Create(path) //nolint:gosec // Path is validated
		if err != nil {
			return nil, nil, err
		}

		return f, WrapCloser(f), nil
	})
}

// OpenFile returns a Resource that opens an existing file at the given path for reading.
// The file is automatically closed when the Resource is used.
func OpenFile(path string) *Resource[*os.File] {
	return NewResource(func() (*os.File, Closer, error) {
		f, err := os.Open(path) //nolint:gosec // Path is validated
		if err != nil {
			return nil, nil, err
		}

		return f, WrapCloser(f), nil
	})
}

// File wraps an existing *os.File as a Resource.
// The file is automatically closed when the Resource is used.
func File(file *os.File) *Resource[*os.File] {
	return NewResource(func() (*os.File, Closer, error) {
		return file, WrapCloser(file), nil
	})
}

// Writer wraps an io.Writer as a Resource.
// If the writer implements io.WriteCloser, it will be closed automatically.
// Otherwise, a no-op closer is used.
func Writer(writer io.Writer) *Resource[io.Writer] {
	wc, ok := writer.(io.WriteCloser)
	if ok {
		return NewResource(func() (io.Writer, Closer, error) {
			return wc, WrapCloser(wc), nil
		})
	} else {
		return NewResource(func() (io.Writer, Closer, error) {
			return writer, func() error {
				return nil
			}, nil
		})
	}
}

// Reader wraps an io.Reader as a Resource.
// If the reader implements io.ReadCloser, it will be closed automatically.
// Otherwise, a no-op closer is used.
func Reader(reader io.Reader) *Resource[io.Reader] {
	rc, ok := reader.(io.ReadCloser)
	if ok {
		return NewResource(func() (io.Reader, Closer, error) {
			return rc, WrapCloser(rc), nil
		})
	} else {
		return NewResource(func() (io.Reader, Closer, error) {
			return reader, func() error {
				return nil
			}, nil
		})
	}
}

// ReadWriter wraps an io.ReadWriter as a Resource.
// If the reader/writer implements io.ReadWriteCloser, it will be closed automatically.
// Otherwise, a no-op closer is used.
func ReadWriter(rw io.ReadWriter) *Resource[io.ReadWriter] {
	rwc, ok := rw.(io.ReadWriteCloser)
	if ok {
		return NewResource(func() (io.ReadWriter, Closer, error) {
			return rwc, WrapCloser(rwc), nil
		})
	} else {
		return NewResource(func() (io.ReadWriter, Closer, error) {
			return rw, func() error {
				return nil
			}, nil
		})
	}
}

// TempDir returns a Resource that creates a temporary directory.
// The directory will be created in the specified dir with the given pattern.
// If dir is empty, the default temporary directory is used.
// The directory and all its contents are automatically removed when the Resource is used.
func TempDir(dir, pattern string) *Resource[string] {
	return NewResource(func() (string, Closer, error) {
		path, err := os.MkdirTemp(dir, pattern)
		if err != nil {
			return "", nil, err
		}

		return path, func() error {
			err := os.RemoveAll(path)
			if err != nil {
				return fmt.Errorf("failed to remove temporary directory %q: %w", path, err)
			}

			return nil
		}, nil
	})
}

// TempFile returns a Resource that creates a temporary file.
// The file will be created in the specified dir with the given pattern.
// If dir is empty, the default temporary directory is used.
// The file is automatically closed and removed when the Resource is used.
func TempFile(dir, pattern string) *Resource[*os.File] {
	return NewResource(func() (*os.File, Closer, error) {
		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, nil, err
		}

		return file, func() error {
			name := file.Name()

			err := file.Close()
			if err != nil {
				return fmt.Errorf("error closing temp file %q: %w", name, err)
			}

			err = os.Remove(name)
			if err != nil {
				return fmt.Errorf("error removing temp file %q: %w", name, err)
			}

			return nil
		}, nil
	})
}

// WrapCloser converts an io.Closer into a Closer function.
// If the closer is nil, it returns a no-op closer that returns nil.
func WrapCloser(closer io.Closer) Closer {
	return func() error {
		if closer != nil {
			return closer.Close()
		}

		return nil
	}
}
