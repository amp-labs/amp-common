package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrepChannel(t *testing.T) {
	t.Parallel()

	t.Run("filters messages matching pattern", func(t *testing.T) {
		t.Parallel()

		w, r, err := GrepChannel("^test")
		require.NoError(t, err)

		go func() {
			w <- "test message 1"
			w <- "other message"
			w <- "test message 2"
			w <- "another message"
			close(w)
		}()

		results := []string{}
		for msg := range r {
			results = append(results, msg)
		}

		assert.Len(t, results, 2)
		assert.Contains(t, results, "test message 1")
		assert.Contains(t, results, "test message 2")
		assert.NotContains(t, results, "other message")
		assert.NotContains(t, results, "another message")
	})

	t.Run("returns all messages when pattern matches everything", func(t *testing.T) {
		t.Parallel()

		w, r, err := GrepChannel(".*")
		require.NoError(t, err)

		go func() {
			w <- "message 1"
			w <- "message 2"
			w <- "message 3"
			close(w)
		}()

		results := []string{}
		for msg := range r {
			results = append(results, msg)
		}

		assert.Len(t, results, 3)
	})

	t.Run("returns no messages when pattern matches nothing", func(t *testing.T) {
		t.Parallel()

		w, r, err := GrepChannel("^nomatch$")
		require.NoError(t, err)

		go func() {
			w <- "message 1"
			w <- "message 2"
			close(w)
		}()

		results := []string{}
		for msg := range r {
			results = append(results, msg)
		}

		assert.Empty(t, results)
	})

	t.Run("closes read channel when write channel is closed", func(t *testing.T) {
		t.Parallel()

		w, r, err := GrepChannel("test")
		require.NoError(t, err)

		close(w)

		// Read channel should be closed
		select {
		case _, ok := <-r:
			assert.False(t, ok, "read channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for channel to close")
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		t.Parallel()

		w, r, err := GrepChannel("test")
		require.NoError(t, err)

		close(w)

		results := []string{}
		for msg := range r {
			results = append(results, msg)
		}

		assert.Empty(t, results)
	})

	t.Run("returns error for invalid regex", func(t *testing.T) {
		t.Parallel()

		_, _, err := GrepChannel("[invalid")
		require.Error(t, err)
	})

	t.Run("filters with case-sensitive pattern", func(t *testing.T) {
		t.Parallel()

		w, r, err := GrepChannel("^Test")
		require.NoError(t, err)

		go func() {
			w <- "Test message"
			w <- "test message"
			close(w)
		}()

		results := []string{}
		for msg := range r {
			results = append(results, msg)
		}

		assert.Len(t, results, 1)
		assert.Equal(t, "Test message", results[0])
	})

	t.Run("handles complex patterns", func(t *testing.T) {
		t.Parallel()

		w, r, err := GrepChannel("[0-9]+")
		require.NoError(t, err)

		go func() {
			w <- "message 123"
			w <- "no numbers here"
			w <- "456 at start"
			close(w)
		}()

		results := []string{}
		for msg := range r {
			results = append(results, msg)
		}

		assert.Len(t, results, 2)
		assert.Contains(t, results, "message 123")
		assert.Contains(t, results, "456 at start")
	})

	t.Run("processes messages in order", func(t *testing.T) {
		t.Parallel()

		w, r, err := GrepChannel("msg")
		require.NoError(t, err)

		go func() {
			w <- "msg1"
			w <- "msg2"
			w <- "msg3"
			close(w)
		}()

		results := []string{}
		for msg := range r {
			results = append(results, msg)
		}

		assert.Equal(t, []string{"msg1", "msg2", "msg3"}, results)
	})
}
