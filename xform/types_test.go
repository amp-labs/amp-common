package xform_test

import (
	"testing"
	"time"

	"github.com/amp-labs/amp-common/xform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	// Test that all error constants are defined and have expected messages
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrInvalidChoice", xform.ErrInvalidChoice, "invalid choice"},
		{"ErrNonPositive", xform.ErrNonPositive, "value must be positive"},
		{"ErrBadPort", xform.ErrBadPort, "invalid port number"},
		{"ErrBadHostAndPort", xform.ErrBadHostAndPort, "invalid host:port pair"},
		{"ErrNotAFile", xform.ErrNotAFile, "not a file"},
		{"ErrNotADir", xform.ErrNotADir, "not a directory"},
		{"ErrEmptyFile", xform.ErrEmptyFile, "empty file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// Test that the type constraints compile and accept the expected types.
func TestNumericConstraint(t *testing.T) {
	t.Parallel()

	// This test verifies that various numeric types work with Positive function
	// which uses the Numeric constraint
	t.Run("int types", func(t *testing.T) {
		t.Parallel()

		_, err := xform.Positive(int(1))
		require.NoError(t, err)

		_, err = xform.Positive(int8(1))
		require.NoError(t, err)

		_, err = xform.Positive(int16(1))
		require.NoError(t, err)

		_, err = xform.Positive(int32(1))
		require.NoError(t, err)

		_, err = xform.Positive(int64(1))
		require.NoError(t, err)
	})

	t.Run("uint types", func(t *testing.T) {
		t.Parallel()

		_, err := xform.Positive(uint(1))
		require.NoError(t, err)

		_, err = xform.Positive(uint8(1))
		require.NoError(t, err)

		_, err = xform.Positive(uint16(1))
		require.NoError(t, err)

		_, err = xform.Positive(uint32(1))
		require.NoError(t, err)

		_, err = xform.Positive(uint64(1))
		require.NoError(t, err)
	})

	t.Run("float types", func(t *testing.T) {
		t.Parallel()

		_, err := xform.Positive(float32(1.0))
		require.NoError(t, err)

		_, err = xform.Positive(float64(1.0))
		require.NoError(t, err)
	})

	t.Run("duration", func(t *testing.T) {
		t.Parallel()

		_, err := xform.Positive(time.Duration(1))
		require.NoError(t, err)
	})
}

func TestCastNumericConstraint(t *testing.T) {
	t.Parallel()

	// Test various numeric type conversions
	t.Run("int to int conversions", func(t *testing.T) {
		t.Parallel()

		_, err := xform.CastNumeric[int64, int32](42)
		require.NoError(t, err)

		_, err = xform.CastNumeric[int32, int64](42)
		require.NoError(t, err)

		_, err = xform.CastNumeric[int, int64](42)
		require.NoError(t, err)
	})

	t.Run("uint to uint conversions", func(t *testing.T) {
		t.Parallel()

		_, err := xform.CastNumeric[uint64, uint32](42)
		require.NoError(t, err)

		_, err = xform.CastNumeric[uint32, uint64](42)
		require.NoError(t, err)
	})

	t.Run("float conversions", func(t *testing.T) {
		t.Parallel()

		_, err := xform.CastNumeric[float64, float32](3.14)
		require.NoError(t, err)

		_, err = xform.CastNumeric[float32, float64](3.14)
		require.NoError(t, err)
	})

	t.Run("mixed type conversions", func(t *testing.T) {
		t.Parallel()

		_, err := xform.CastNumeric[int, float64](42)
		require.NoError(t, err)

		_, err = xform.CastNumeric[float64, int](42.7)
		require.NoError(t, err)

		_, err = xform.CastNumeric[uint, int](42)
		require.NoError(t, err)
	})

	t.Run("duration conversions", func(t *testing.T) {
		t.Parallel()

		_, err := xform.CastNumeric[int64, time.Duration](1000000000)
		require.NoError(t, err)

		_, err = xform.CastNumeric[time.Duration, int64](time.Second)
		require.NoError(t, err)
	})
}
