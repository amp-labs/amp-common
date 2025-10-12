// Package closer provides utilities for managing io.Closer resources.
//
// The package includes:
//   - Closer: A collector that manages multiple io.Closer instances and closes them all at once
//   - CloseOnce: A thread-safe wrapper that ensures an io.Closer is only closed once
//   - HandlePanic: A wrapper that recovers from panics in Close() and converts them to errors
//   - ChannelCloser: A generic io.Closer wrapper for channels
//   - CustomCloser: Creates an io.Closer from any cleanup function
package closer

import (
	"errors"
	"io"
	"runtime/debug"
	"sync"

	"github.com/amp-labs/amp-common/utils"
	"go.uber.org/atomic"
)

// customCloser is an internal implementation that wraps a function to make it an io.Closer.
// This allows any cleanup function to be used as an io.Closer, enabling it to work with
// utilities like Closer, CloseOnce, and HandlePanic.
type customCloser struct {
	closeFn func() error // The cleanup function to execute when Close() is called
}

// CustomCloser creates an io.Closer from a cleanup function.
// This allows arbitrary cleanup logic to be integrated with the io.Closer interface.
//
// Special cases:
//   - Returns nil if closeFn is nil
//
// Example usage:
//
//	cleanup := func() error {
//	    // Custom cleanup logic
//	    return disconnectDatabase()
//	}
//	closer := CustomCloser(cleanup)
//	defer closer.Close()
//
// Example with Closer collector:
//
//	collector := NewCloser()
//	collector.Add(CustomCloser(func() error {
//	    log.Println("cleanup step 1")
//	    return nil
//	}))
//	collector.Add(CustomCloser(func() error {
//	    log.Println("cleanup step 2")
//	    return nil
//	}))
//	defer collector.Close()
func CustomCloser(closeFn func() error) io.Closer {
	if closeFn == nil {
		return nil
	}

	return &customCloser{closeFn: closeFn}
}

// Close executes the wrapped cleanup function and returns its result.
func (c *customCloser) Close() error {
	if c.closeFn != nil {
		return c.closeFn()
	}

	return nil
}

// Closer is a collector that manages multiple io.Closer instances.
// It allows you to add closers incrementally and close them all at once,
// collecting any errors that occur during the close operations.
//
// Example usage:
//
//	closer := NewCloser()
//	file, err := os.Open("example.txt")
//	if err != nil {
//	    return err
//	}
//	closer.Add(file)
//
//	conn, err := net.Dial("tcp", "example.com:80")
//	if err != nil {
//	    closer.Close() // Close file if connection fails
//	    return err
//	}
//	closer.Add(conn)
//
//	// Both file and conn will be closed, even if one returns an error
//	return closer.Close()
type Closer struct {
	closers []io.Closer
}

// NewCloser creates a new Closer with zero or more initial io.Closer instances.
//
// Example:
//
//	closer := NewCloser(file1, file2, conn)
//	defer closer.Close()
func NewCloser(closers ...io.Closer) *Closer {
	return &Closer{closers: closers}
}

// Add adds an io.Closer to the collection. The closer will be closed when Close() is called.
// Nil closers are allowed and will be safely skipped during Close().
//
// Note: Add is not thread-safe. If you need to add closers concurrently, use external synchronization.
func (c *Closer) Add(closer io.Closer) {
	c.closers = append(c.closers, closer)
}

// Close closes all registered io.Closer instances in the order they were added.
// If any closers return errors, all closers will still be attempted, and all errors
// will be collected and returned as a joined error using errors.Join.
//
// Nil closers are safely skipped.
//
// Returns nil if all closers succeeded, or a joined error containing all failures.
func (c *Closer) Close() error {
	var errs []error

	for _, closer := range c.closers {
		if closer != nil {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// closeOnceImpl is the internal implementation of a thread-safe single-close wrapper.
// It uses a mutex to ensure that only one Close() call actually invokes the underlying closer.
type closeOnceImpl struct {
	mut    sync.Mutex
	closed bool      // Protected by mut: tracks whether Close() has completed successfully
	closer io.Closer // The underlying closer to protect
}

// CloseOnce wraps an io.Closer to ensure it can only be closed once.
// Subsequent calls to Close() will be no-ops and return nil.
//
// This is useful for resources that may be shared across multiple goroutines or
// passed through multiple cleanup paths, where you want to ensure Close() is called
// but avoid double-close bugs.
//
// Thread-safety: CloseOnce is safe for concurrent use. Multiple goroutines can
// call Close() simultaneously, and only one will actually close the underlying resource.
//
// Error handling: If the underlying Close() returns an error, the resource is NOT
// marked as closed, and subsequent Close() calls will retry. This ensures that
// transient errors can be retried, but successful closes are idempotent.
//
// Special cases:
//   - Returns nil if the input closer is nil
//   - If the input is already a *closeOnceImpl, returns it unchanged (idempotent)
//
// Example usage:
//
//	closer := CloseOnce(file)
//	defer closer.Close()  // Safe even if explicitly closed elsewhere
//
//	// ... later in code ...
//	if someCondition {
//	    closer.Close()  // Won't double-close
//	}
func CloseOnce(closer io.Closer) io.Closer {
	if closer == nil {
		return nil
	}

	// Idempotent: if already wrapped, return the existing wrapper
	once, ok := closer.(*closeOnceImpl)
	if ok {
		return once
	}

	return &closeOnceImpl{closer: closer}
}

// Close closes the underlying io.Closer exactly once. Subsequent calls return nil without closing.
//
// If the first Close() call returns an error, the closer is NOT marked as closed,
// allowing subsequent Close() calls to retry. This is intentional to handle transient errors.
// Once Close() succeeds (returns nil), all future calls will return nil without invoking
// the underlying closer.
//
// Thread-safety: This method is safe for concurrent use.
func (c *closeOnceImpl) Close() error {
	if c.closer == nil {
		return nil
	}

	c.mut.Lock()
	defer c.mut.Unlock()

	if c.closed {
		return nil
	}

	if err := c.closer.Close(); err != nil {
		return err
	}

	c.closed = true

	return nil
}

// HandlePanic wraps an io.Closer to recover from panics during Close() and convert them to errors.
// This prevents panics in Close() calls from crashing the program, which is especially useful
// when closing resources in cleanup code or deferred statements.
//
// If the wrapped closer panics during Close(), the panic is recovered and converted to an error
// that includes the panic value and stack trace. If Close() also returns an error, both the
// panic error and the Close() error are joined using errors.Join.
//
// Thread-safety: HandlePanic wrappers are safe for concurrent use if the underlying closer is.
// However, like most io.Closer implementations, calling Close() concurrently is not recommended
// unless the underlying closer explicitly supports it.
//
// Special cases:
//   - Returns nil if the input closer is nil
//   - If the input is already a *panicHandlingImpl, returns it unchanged (idempotent)
//
// Example usage:
//
//	closer := HandlePanic(riskyCloser)
//	if err := closer.Close(); err != nil {
//	    // Will receive an error instead of a panic if riskyCloser panics
//	    log.Printf("close failed: %v", err)
//	}
func HandlePanic(closer io.Closer) io.Closer {
	if closer == nil {
		return nil
	}

	// Idempotent: if already wrapped, return the existing wrapper
	if _, ok := closer.(*panicHandlingImpl); ok {
		return closer
	}

	return &panicHandlingImpl{closer: closer}
}

// panicHandlingImpl is the internal implementation of a panic-recovering closer wrapper.
// It uses defer/recover to catch panics from the underlying closer's Close() method.
type panicHandlingImpl struct {
	closer io.Closer // The underlying closer to protect
}

// Close calls the underlying closer's Close() method with panic recovery.
// If the underlying Close() panics, the panic is recovered and converted to an error.
// If both Close() returns an error AND panics, both errors are joined.
func (p *panicHandlingImpl) Close() (err error) {
	if p.closer == nil {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err2 := utils.GetPanicRecoveryError(r, debug.Stack())
			if err == nil {
				err = err2
			} else {
				err = errors.Join(err, err2)
			}
		}
	}()

	return p.closer.Close()
}

// channelCloserImpl is a generic io.Closer implementation for closing channels.
// It accepts a send-only channel since only senders should close channels.
type channelCloserImpl[T any] struct {
	ch chan<- T
}

// ChannelCloser wraps a send-only channel and returns an io.Closer that will close the channel when Close() is called.
// Accepts send-only channels (chan<-) since only the sender should close a channel in Go.
//
// This is useful when you want to manage channel lifecycle using the io.Closer interface,
// allowing channels to be used with utilities like Closer, CloseOnce, and HandlePanic.
//
// Thread-safety: Closing a channel is not inherently thread-safe in Go. If you need to close
// a channel from multiple goroutines, wrap the result with CloseOnce:
//
//	closer := CloseOnce(ChannelCloser(ch))
//
// Panic handling: If you need to handle panics from closing an already-closed channel,
// wrap the result with HandlePanic:
//
//	closer := HandlePanic(ChannelCloser(ch))
//
// Or combine both for thread-safe panic handling:
//
//	closer := HandlePanic(CloseOnce(ChannelCloser(ch)))
//
// Special cases:
//   - Returns nil if the input channel is nil
//   - Will panic if the channel is already closed (use HandlePanic to prevent this)
//   - Accepts send-only channels (chan<-) or bidirectional channels (chan) which implicitly convert to chan<-
//
// Example usage:
//
//	ch := make(chan int)
//	closer := ChannelCloser(ch)
//	defer closer.Close()
//
//	// Use channel...
//
//	// Channel will be closed when closer.Close() is called
//
// Example with send-only channel:
//
//	func worker(ch chan<- int, closer io.Closer) {
//	    defer closer.Close()
//	    ch <- 42
//	}
//
//	ch := make(chan int)
//	go worker(ch, ChannelCloser(ch))
//
// Example with CloseOnce for thread-safety:
//
//	ch := make(chan string)
//	closer := CloseOnce(ChannelCloser(ch))
//
//	// Safe to call from multiple goroutines
//	go func() { closer.Close() }()
//	go func() { closer.Close() }()
//
// Example with Closer collector:
//
//	collector := NewCloser()
//	ch1 := make(chan int)
//	ch2 := make(chan string)
//	collector.Add(ChannelCloser(ch1))
//	collector.Add(ChannelCloser(ch2))
//	defer collector.Close() // Both channels will be closed
func ChannelCloser[T any](ch chan<- T) io.Closer {
	if ch == nil {
		return nil
	}

	return &channelCloserImpl[T]{ch: ch}
}

// Close closes the wrapped channel.
// Will panic if the channel is already closed (use HandlePanic wrapper to prevent this).
func (c *channelCloserImpl[T]) Close() error {
	if c.ch == nil {
		return nil
	}

	close(c.ch)

	return nil
}

// cancelableCloser is an internal implementation that wraps an io.Closer with the ability
// to cancel the close operation. It uses an atomic boolean to track whether Close() should
// actually close the underlying resource or be a no-op.
type cancelableCloser struct {
	shouldClose *atomic.Bool // Atomic flag: true means Close() will close, false means Close() is a no-op
	closer      io.Closer    // The underlying closer to conditionally close
}

// Close conditionally closes the underlying io.Closer based on the shouldClose flag.
// If the cancel function has been called, this method returns nil without closing.
// Otherwise, it closes the underlying closer and returns any error.
//
// Thread-safety: This method is safe for concurrent use due to the atomic shouldClose flag.
func (c *cancelableCloser) Close() error {
	if c.closer == nil {
		return nil
	}

	if c.shouldClose.Load() {
		return c.closer.Close()
	}

	return nil
}

// cancel prevents future Close() calls from closing the underlying resource.
// After calling cancel(), Close() will become a no-op that returns nil.
// This method is safe for concurrent use.
func (c *cancelableCloser) cancel() {
	c.shouldClose.Store(false)
}

// CancelableCloser wraps an io.Closer with the ability to cancel the close operation.
// It returns both a closer and a cancel function. If the cancel function is called before Close(),
// then Close() will become a no-op and return nil without closing the underlying resource.
//
// This is useful for resource management scenarios where you want to conditionally clean up
// based on success/failure, such as:
//   - Transaction-like behavior (commit on success, rollback on failure)
//   - Temporary file handling (delete on error, keep on success)
//   - Connection pooling (return to pool on success, close on error)
//
// Thread-safety: Both the returned closer and cancel function are safe for concurrent use.
// Multiple goroutines can call Close() and cancel() simultaneously.
//
// Special cases:
//   - Returns (nil, no-op function) if the input closer is nil
//   - If the input is already a *cancelableCloser, returns it with its cancel function (idempotent)
//
// Example usage - temporary file handling:
//
//	tmpFile, err := os.CreateTemp("", "example")
//	if err != nil {
//	    return err
//	}
//	closer, cancel := CancelableCloser(CustomCloser(func() error {
//	    tmpFile.Close()
//	    return os.Remove(tmpFile.Name())
//	}))
//	defer closer.Close() // Will delete file unless cancel() is called
//
//	// ... process file ...
//
//	if success {
//	    cancel() // Keep the file, don't delete it
//	}
//
// Example usage - transaction pattern:
//
//	tx, err := db.Begin()
//	if err != nil {
//	    return err
//	}
//	closer, cancel := CancelableCloser(CustomCloser(func() error {
//	    return tx.Rollback() // Rollback unless canceled
//	}))
//	defer closer.Close()
//
//	// ... do work ...
//
//	if err := tx.Commit(); err != nil {
//	    return err // Rollback via deferred Close()
//	}
//	cancel() // Success, don't rollback
//	return nil
func CancelableCloser(c io.Closer) (closer io.Closer, cancel func()) {
	if c == nil {
		return nil, func() {}
	}

	cc, ok := c.(*cancelableCloser)
	if ok {
		return cc, cc.cancel
	}

	cc = &cancelableCloser{
		shouldClose: atomic.NewBool(true),
		closer:      c,
	}

	return cc, cc.cancel
}
