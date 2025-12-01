package envutil_test

import (
	"testing"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestCombine2(t *testing.T) {
	t.Run("combines two present values", func(t *testing.T) {
		t.Setenv("TEST_COMBINE2_A", "value1")
		t.Setenv("TEST_COMBINE2_B", "value2")

		reader := envutil.String2(t.Context(), "TEST_COMBINE2_A", "TEST_COMBINE2_B")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", value.First())
		assert.Equal(t, "value2", value.Second())
	})

	t.Run("fails when first is missing", func(t *testing.T) {
		t.Setenv("TEST_COMBINE2_MISSING_B", "value2")

		reader := envutil.String2(t.Context(), "TEST_COMBINE2_MISSING_A", "TEST_COMBINE2_MISSING_B")
		_, err := reader.Value()
		require.Error(t, err)
	})

	t.Run("fails when second is missing", func(t *testing.T) {
		t.Setenv("TEST_COMBINE2_MISSING_A", "value1")

		reader := envutil.String2(t.Context(), "TEST_COMBINE2_MISSING_A", "TEST_COMBINE2_MISSING_B")
		_, err := reader.Value()
		require.Error(t, err)
	})

	t.Run("fails when both are missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String2(t.Context(), "TEST_COMBINE2_MISSING_A", "TEST_COMBINE2_MISSING_B")
		_, err := reader.Value()
		require.Error(t, err)
	})

	t.Run("fails when first has error", func(t *testing.T) {
		t.Setenv("TEST_COMBINE2_ERROR_A", "not-a-number")
		t.Setenv("TEST_COMBINE2_ERROR_B", "42")

		reader := envutil.Combine2(
			envutil.Int[int](t.Context(), "TEST_COMBINE2_ERROR_A"),
			envutil.Int[int](t.Context(), "TEST_COMBINE2_ERROR_B"),
		)
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestCombine3(t *testing.T) {
	t.Run("combines three present values", func(t *testing.T) {
		t.Setenv("TEST_COMBINE3_A", "value1")
		t.Setenv("TEST_COMBINE3_B", "value2")
		t.Setenv("TEST_COMBINE3_C", "value3")

		reader := envutil.String3(t.Context(), "TEST_COMBINE3_A", "TEST_COMBINE3_B", "TEST_COMBINE3_C")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", value.First())
		assert.Equal(t, "value2", value.Second())
		assert.Equal(t, "value3", value.Third())
	})

	t.Run("fails when any is missing", func(t *testing.T) {
		t.Setenv("TEST_COMBINE3_MISSING_A", "value1")
		t.Setenv("TEST_COMBINE3_MISSING_B", "value2")

		reader := envutil.String3(t.Context(),
			"TEST_COMBINE3_MISSING_A", "TEST_COMBINE3_MISSING_B", "TEST_COMBINE3_MISSING_C")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestCombine4(t *testing.T) {
	t.Run("combines four present values", func(t *testing.T) {
		t.Setenv("TEST_COMBINE4_A", "value1")
		t.Setenv("TEST_COMBINE4_B", "value2")
		t.Setenv("TEST_COMBINE4_C", "value3")
		t.Setenv("TEST_COMBINE4_D", "value4")

		reader := envutil.String4(t.Context(), "TEST_COMBINE4_A", "TEST_COMBINE4_B", "TEST_COMBINE4_C", "TEST_COMBINE4_D")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", value.First())
		assert.Equal(t, "value2", value.Second())
		assert.Equal(t, "value3", value.Third())
		assert.Equal(t, "value4", value.Fourth())
	})

	t.Run("fails when any is missing", func(t *testing.T) {
		t.Setenv("TEST_COMBINE4_MISSING_A", "value1")

		reader := envutil.String4(
			t.Context(),
			"TEST_COMBINE4_MISSING_A",
			"TEST_COMBINE4_MISSING_B",
			"TEST_COMBINE4_MISSING_C",
			"TEST_COMBINE4_MISSING_D",
		)
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestCombine5(t *testing.T) {
	t.Run("combines five present values", func(t *testing.T) {
		t.Setenv("TEST_COMBINE5_A", "value1")
		t.Setenv("TEST_COMBINE5_B", "value2")
		t.Setenv("TEST_COMBINE5_C", "value3")
		t.Setenv("TEST_COMBINE5_D", "value4")
		t.Setenv("TEST_COMBINE5_E", "value5")

		reader := envutil.String5(
			t.Context(),
			"TEST_COMBINE5_A",
			"TEST_COMBINE5_B",
			"TEST_COMBINE5_C",
			"TEST_COMBINE5_D",
			"TEST_COMBINE5_E",
		)
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", value.First())
		assert.Equal(t, "value2", value.Second())
		assert.Equal(t, "value3", value.Third())
		assert.Equal(t, "value4", value.Fourth())
		assert.Equal(t, "value5", value.Fifth())
	})
}

func TestCombine6(t *testing.T) {
	t.Run("combines six present values", func(t *testing.T) {
		t.Setenv("TEST_COMBINE6_A", "value1")
		t.Setenv("TEST_COMBINE6_B", "value2")
		t.Setenv("TEST_COMBINE6_C", "value3")
		t.Setenv("TEST_COMBINE6_D", "value4")
		t.Setenv("TEST_COMBINE6_E", "value5")
		t.Setenv("TEST_COMBINE6_F", "value6")

		reader := envutil.String6(
			t.Context(),
			"TEST_COMBINE6_A",
			"TEST_COMBINE6_B",
			"TEST_COMBINE6_C",
			"TEST_COMBINE6_D",
			"TEST_COMBINE6_E",
			"TEST_COMBINE6_F",
		)
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", value.First())
		assert.Equal(t, "value2", value.Second())
		assert.Equal(t, "value3", value.Third())
		assert.Equal(t, "value4", value.Fourth())
		assert.Equal(t, "value5", value.Fifth())
		assert.Equal(t, "value6", value.Sixth())
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestSplit2(t *testing.T) {
	t.Run("splits tuple into readers", func(t *testing.T) {
		t.Setenv("TEST_SPLIT2_A", "value1")
		t.Setenv("TEST_SPLIT2_B", "value2")

		combined := envutil.String2(t.Context(), "TEST_SPLIT2_A", "TEST_SPLIT2_B")
		first, second := envutil.Split2(combined)

		val1, err := first.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := second.Value()
		require.NoError(t, err)
		assert.Equal(t, "value2", val2)
	})

	t.Run("propagates missing state", func(t *testing.T) {
		t.Parallel()

		combined := envutil.String2(t.Context(), "TEST_SPLIT2_MISSING_A", "TEST_SPLIT2_MISSING_B")
		first, second := envutil.Split2(combined)

		_, err := first.Value()
		require.Error(t, err)

		_, err = second.Value()
		require.Error(t, err)
	})

	t.Run("propagates error state", func(t *testing.T) {
		t.Setenv("TEST_SPLIT2_ERROR_A", "not-a-number")
		t.Setenv("TEST_SPLIT2_ERROR_B", "42")

		combined := envutil.Combine2(
			envutil.Int[int](t.Context(), "TEST_SPLIT2_ERROR_A"),
			envutil.Int[int](t.Context(), "TEST_SPLIT2_ERROR_B"),
		)
		first, second := envutil.Split2(combined)

		_, err := first.Value()
		require.Error(t, err)

		_, err = second.Value()
		require.Error(t, err)
	})
}

func TestSplit3(t *testing.T) {
	t.Run("splits tuple into readers", func(t *testing.T) {
		t.Setenv("TEST_SPLIT3_A", "value1")
		t.Setenv("TEST_SPLIT3_B", "value2")
		t.Setenv("TEST_SPLIT3_C", "value3")

		combined := envutil.String3(t.Context(), "TEST_SPLIT3_A", "TEST_SPLIT3_B", "TEST_SPLIT3_C")
		first, second, third := envutil.Split3(combined)

		val1, err := first.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := second.Value()
		require.NoError(t, err)
		assert.Equal(t, "value2", val2)

		val3, err := third.Value()
		require.NoError(t, err)
		assert.Equal(t, "value3", val3)
	})
}

func TestSplit4(t *testing.T) {
	t.Run("splits tuple into readers", func(t *testing.T) {
		t.Setenv("TEST_SPLIT4_A", "value1")
		t.Setenv("TEST_SPLIT4_B", "value2")
		t.Setenv("TEST_SPLIT4_C", "value3")
		t.Setenv("TEST_SPLIT4_D", "value4")

		combined := envutil.String4(t.Context(), "TEST_SPLIT4_A", "TEST_SPLIT4_B", "TEST_SPLIT4_C", "TEST_SPLIT4_D")
		first, second, third, fourth := envutil.Split4(combined)

		val1, err := first.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := second.Value()
		require.NoError(t, err)
		assert.Equal(t, "value2", val2)

		val3, err := third.Value()
		require.NoError(t, err)
		assert.Equal(t, "value3", val3)

		val4, err := fourth.Value()
		require.NoError(t, err)
		assert.Equal(t, "value4", val4)
	})
}

func TestSplit5(t *testing.T) {
	t.Run("splits tuple into readers", func(t *testing.T) {
		t.Setenv("TEST_SPLIT5_A", "value1")
		t.Setenv("TEST_SPLIT5_B", "value2")
		t.Setenv("TEST_SPLIT5_C", "value3")
		t.Setenv("TEST_SPLIT5_D", "value4")
		t.Setenv("TEST_SPLIT5_E", "value5")

		combined := envutil.String5(
			t.Context(),
			"TEST_SPLIT5_A",
			"TEST_SPLIT5_B",
			"TEST_SPLIT5_C",
			"TEST_SPLIT5_D",
			"TEST_SPLIT5_E",
		)
		first, second, third, fourth, fifth := envutil.Split5(combined)

		val1, err := first.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := second.Value()
		require.NoError(t, err)
		assert.Equal(t, "value2", val2)

		val3, err := third.Value()
		require.NoError(t, err)
		assert.Equal(t, "value3", val3)

		val4, err := fourth.Value()
		require.NoError(t, err)
		assert.Equal(t, "value4", val4)

		val5, err := fifth.Value()
		require.NoError(t, err)
		assert.Equal(t, "value5", val5)
	})
}

func TestSplit6(t *testing.T) {
	t.Run("splits tuple into readers", func(t *testing.T) {
		t.Setenv("TEST_SPLIT6_A", "value1")
		t.Setenv("TEST_SPLIT6_B", "value2")
		t.Setenv("TEST_SPLIT6_C", "value3")
		t.Setenv("TEST_SPLIT6_D", "value4")
		t.Setenv("TEST_SPLIT6_E", "value5")
		t.Setenv("TEST_SPLIT6_F", "value6")

		combined := envutil.String6(
			t.Context(),
			"TEST_SPLIT6_A",
			"TEST_SPLIT6_B",
			"TEST_SPLIT6_C",
			"TEST_SPLIT6_D",
			"TEST_SPLIT6_E",
			"TEST_SPLIT6_F",
		)
		first, second, third, fourth, fifth, sixth := envutil.Split6(combined)

		val1, err := first.Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := second.Value()
		require.NoError(t, err)
		assert.Equal(t, "value2", val2)

		val3, err := third.Value()
		require.NoError(t, err)
		assert.Equal(t, "value3", val3)

		val4, err := fourth.Value()
		require.NoError(t, err)
		assert.Equal(t, "value4", val4)

		val5, err := fifth.Value()
		require.NoError(t, err)
		assert.Equal(t, "value5", val5)

		val6, err := sixth.Value()
		require.NoError(t, err)
		assert.Equal(t, "value6", val6)
	})
}
