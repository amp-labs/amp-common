package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCloser is a simple mock object that implements io.Closer.
type mockCloser struct {
	id          int
	closeErr    error
	closeCalled *atomic.Bool
}

func newMockCloser(id int) *mockCloser {
	return &mockCloser{
		id:          id,
		closeCalled: &atomic.Bool{},
	}
}

func (m *mockCloser) Close() error {
	m.closeCalled.Store(true)

	return m.closeErr
}

func (m *mockCloser) IsClosed() bool {
	return m.closeCalled.Load()
}

// mockFactory creates mock closers with a counter.
type mockFactory struct {
	counter   *atomic.Int64
	createErr error
	mu        sync.Mutex
	created   []*mockCloser
}

func newMockFactory() *mockFactory {
	return &mockFactory{
		counter: &atomic.Int64{},
		created: make([]*mockCloser, 0),
	}
}

func (f *mockFactory) create() (*mockCloser, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}

	id := int(f.counter.Add(1))
	obj := newMockCloser(id)

	f.mu.Lock()
	f.created = append(f.created, obj)
	f.mu.Unlock()

	return obj, nil
}

func (f *mockFactory) CreatedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.created)
}

func (f *mockFactory) GetCreated(index int) *mockCloser {
	f.mu.Lock()
	defer f.mu.Unlock()

	if index < 0 || index >= len(f.created) {
		return nil
	}

	return f.created[index]
}

func TestNew(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	require.NotNil(t, pool)

	err := pool.Close()
	require.NoError(t, err)
}

func TestGetAndPut(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	defer func() {
		_ = pool.Close()
	}()

	// Get first object - should create new one
	obj1, err := pool.Get()
	require.NoError(t, err)
	require.NotNil(t, obj1)
	assert.Equal(t, 1, factory.CreatedCount())

	// Put it back
	pool.Put(obj1)

	// Small delay to ensure Put completes
	time.Sleep(10 * time.Millisecond)

	// Get again - should reuse the same object
	obj2, err := pool.Get()
	require.NoError(t, err)
	require.NotNil(t, obj2)
	assert.Equal(t, 1, factory.CreatedCount(), "should reuse existing object")
	assert.Equal(t, obj1.id, obj2.id)

	// Put it back so Close doesn't hang
	pool.Put(obj2)
	time.Sleep(10 * time.Millisecond)
}

func TestPoolGrowth(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	defer func() {
		_ = pool.Close()
	}()

	// Get multiple objects without putting them back
	obj1, err := pool.Get()
	require.NoError(t, err)
	require.NotNil(t, obj1)

	obj2, err := pool.Get()
	require.NoError(t, err)
	require.NotNil(t, obj2)

	obj3, err := pool.Get()
	require.NoError(t, err)
	require.NotNil(t, obj3)

	assert.Equal(t, 3, factory.CreatedCount())
	assert.NotEqual(t, obj1.id, obj2.id)
	assert.NotEqual(t, obj2.id, obj3.id)

	// Put them back so Close doesn't hang
	pool.Put(obj1)
	pool.Put(obj2)
	pool.Put(obj3)
	time.Sleep(10 * time.Millisecond)
}

func TestCloseIdle(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	defer func() {
		_ = pool.Close()
	}()

	// Get multiple objects at once (so they don't get reused)
	obj1, err := pool.Get()
	require.NoError(t, err)

	obj2, err := pool.Get()
	require.NoError(t, err)

	// Put them back
	pool.Put(obj1)
	pool.Put(obj2)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Close idle objects that have been idle for less than 1ms (should close all)
	closed, err := pool.CloseIdle(1 * time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, 2, closed)

	// Verify all created objects were closed
	for i := range factory.CreatedCount() {
		obj := factory.GetCreated(i)
		if obj != nil {
			assert.True(t, obj.IsClosed())
		}
	}
}

func TestCloseIdleMinTime(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	defer func() {
		_ = pool.Close()
	}()

	// Create and return object
	obj1, err := pool.Get()
	require.NoError(t, err)
	pool.Put(obj1)

	time.Sleep(10 * time.Millisecond)

	// Try to close idle with high minimum time - shouldn't close anything
	closed, err := pool.CloseIdle(1 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 0, closed)

	// Object should still be alive
	assert.False(t, factory.GetCreated(0).IsClosed())
}

func TestClose(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	// Get multiple objects at once (so they don't get reused)
	obj1, err := pool.Get()
	require.NoError(t, err)

	obj2, err := pool.Get()
	require.NoError(t, err)

	// Put them back
	pool.Put(obj1)
	pool.Put(obj2)

	time.Sleep(10 * time.Millisecond)

	// Close pool
	err = pool.Close()
	require.NoError(t, err)

	// All created objects should be closed
	for i := range factory.CreatedCount() {
		obj := factory.GetCreated(i)
		if obj != nil {
			assert.True(t, obj.IsClosed())
		}
	}
}

var errFactory = errors.New("factory error")

func TestFactoryError(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	factory.createErr = errFactory

	pool := New(factory.create, WithName("test-pool"))
	defer func() {
		_ = pool.Close()
	}()

	obj, err := pool.Get()
	require.Error(t, err)
	assert.Nil(t, obj)
	assert.Equal(t, "factory error", err.Error())
}

var errClose = errors.New("close error")

func TestCloseError(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	// Get an object
	obj, err := pool.Get()
	require.NoError(t, err)

	// Set it to return error on close
	obj.closeErr = errClose

	pool.Put(obj)
	time.Sleep(10 * time.Millisecond)

	// CloseIdle should return the error
	_, err = pool.CloseIdle(1 * time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close error")
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	defer func() {
		_ = pool.Close()
	}()

	const (
		goroutines = 10
		iterations = 20
	)

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	for range goroutines {
		go func() {
			defer waitGroup.Done()

			for range iterations {
				obj, err := pool.Get()
				if err != nil {
					continue
				}

				// Simulate some work
				time.Sleep(1 * time.Millisecond)

				pool.Put(obj)
			}
		}()
	}

	waitGroup.Wait()

	// Pool should have created some objects (but likely fewer than total gets due to reuse)
	assert.Positive(t, factory.CreatedCount())
	assert.Less(t, factory.CreatedCount(), goroutines*iterations)
}

func TestErrPoolClosed(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	// Close the pool
	err := pool.Close()
	require.NoError(t, err)

	// Try to get from closed pool
	obj, err := pool.Get()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPoolClosed)
	assert.Nil(t, obj)

	// Try to put to closed pool (should log error but not panic)
	mockObj := newMockCloser(999)
	pool.Put(mockObj)

	// Try to close idle on closed pool
	closed, err := pool.CloseIdle(1 * time.Millisecond)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPoolClosed)
	assert.Equal(t, 0, closed)

	// Try to close again
	err = pool.Close()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPoolClosed)
}

func TestPoolWithName(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("custom-name"))

	defer func() {
		_ = pool.Close()
	}()

	poolImpl, ok := pool.(*poolImpl[*mockCloser])
	require.True(t, ok)
	assert.Equal(t, "custom-name", poolImpl.name)
}

func TestPoolDefaultName(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create)

	defer func() {
		_ = pool.Close()
	}()

	poolImpl, ok := pool.(*poolImpl[*mockCloser])
	require.True(t, ok)
	assert.Equal(t, "pool", poolImpl.name)
}

func TestMultipleGetsPutsReuse(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	defer func() {
		_ = pool.Close()
	}()

	// Get 3 objects
	objs := make([]*mockCloser, 3)

	for i := range objs {
		obj, err := pool.Get()
		require.NoError(t, err)

		objs[i] = obj
	}

	assert.Equal(t, 3, factory.CreatedCount())

	// Put them all back
	for _, obj := range objs {
		pool.Put(obj)
	}

	time.Sleep(10 * time.Millisecond)

	// Get 3 more - should reuse existing ones
	moreObjs := make([]*mockCloser, 3)

	for i := range moreObjs {
		obj, err := pool.Get()
		require.NoError(t, err)

		moreObjs[i] = obj
	}

	// Should still only have 3 created objects
	assert.Equal(t, 3, factory.CreatedCount())

	// Put them back so Close doesn't hang
	for _, obj := range moreObjs {
		pool.Put(obj)
	}

	time.Sleep(10 * time.Millisecond)
}

func TestCloseWithOutstandingObjects(t *testing.T) {
	t.Parallel()

	factory := newMockFactory()
	pool := New(factory.create, WithName("test-pool"))

	// Get an object but don't put it back
	obj, err := pool.Get()
	require.NoError(t, err)
	require.NotNil(t, obj)

	// Close the pool - it should wait for outstanding objects
	errChan := make(chan error, 1)
	go func() {
		errChan <- pool.Close()
	}()

	// Give it a moment
	time.Sleep(50 * time.Millisecond)

	// Put the object back
	pool.Put(obj)

	// Now Close should complete
	select {
	case err := <-errChan:
		require.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("Close did not complete after returning object")
	}
}
