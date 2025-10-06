package xform_test

import (
	"testing"

	"github.com/amp-labs/amp-common/xform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrInvalidGzipLevel(t *testing.T) {
	t.Parallel()

	// Test that the error is defined and has expected message
	require.Error(t, xform.ErrInvalidGzipLevel)
	assert.Equal(t, "invalid gzip level", xform.ErrInvalidGzipLevel.Error())
}
