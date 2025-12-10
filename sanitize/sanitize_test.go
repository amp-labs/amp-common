package sanitize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple ascii filename",
			input:    "test-file.txt",
			expected: "test-file.txt",
		},
		{
			name:     "german umlauts lowercase",
			input:    "äöü",
			expected: "aeoeue",
		},
		{
			name:     "german umlauts uppercase",
			input:    "ÄÖÜ",
			expected: "AeOeUe",
		},
		{
			name:     "german sharp s",
			input:    "Straße",
			expected: "Strasse",
		},
		{
			name:     "french cedilla",
			input:    "français",
			expected: "francais",
		},
		{
			name:     "ampersand replacement",
			input:    "rock&roll",
			expected: "rock_and_roll",
		},
		{
			name:     "plus replacement",
			input:    "C++",
			expected: "C_plus_plus_",
		},
		{
			name:     "at symbol replacement",
			input:    "user@email",
			expected: "user_at_email",
		},
		{
			name:     "currency symbols",
			input:    "€100-£50-$25-¥1000",
			expected: "Euro100-Pound50-Dollar25-Yen1000",
		},
		{
			name:     "forbidden characters windows",
			input:    `file:name/path\test`,
			expected: "file_name_path_test",
		},
		{
			name:     "special shell characters",
			input:    "file name (test)",
			expected: "file_name_test_",
		},
		{
			name:     "quotes and brackets",
			input:    `"file"[test]{name}`,
			expected: "_file_test_name_",
		},
		{
			name:     "wildcards and pipes",
			input:    "file*name?test|data",
			expected: "file_name_test_data",
		},
		{
			name:     "whitespace characters",
			input:    "file\nname\ttab\rreturn",
			expected: "file_name_tab_return",
		},
		{
			name:     "special symbols",
			input:    "test!#%<>~^'`°§",
			expected: "test_",
		},
		{
			name:     "multiple consecutive underscores collapsed",
			input:    "file   name",
			expected: "file_name",
		},
		{
			name:     "accented characters removed",
			input:    "café",
			expected: "cafe",
		},
		{
			name:     "leading dash trimmed",
			input:    "-filename",
			expected: "filename",
		},
		{
			name:     "trailing dash trimmed",
			input:    "filename-",
			expected: "filename",
		},
		{
			name:     "both leading and trailing dashes trimmed",
			input:    "-filename-",
			expected: "filename",
		},
		{
			name:     "non-ascii characters replaced",
			input:    "test文件名", //nolint:gosmopolitan // Intentional test data for non-ASCII handling
			expected: "test_",
		},
		{
			name:     "complex mixed input",
			input:    "Müller & Söhne (€100).txt",
			expected: "Mueller_and_Soehne_Euro100_.txt",
		},
		{
			name: "all forbidden characters",
			input: `":/\()?*
 {|}[¦]!#%<>~^'` + "`°§",
			expected: "_",
		},
		{
			name:     "dots are preserved",
			input:    "file.name.test.txt",
			expected: "file.name.test.txt",
		},
		{
			name:     "numbers preserved",
			input:    "file123name456",
			expected: "file123name456",
		},
		{
			name:     "underscore preserved",
			input:    "file_name_test",
			expected: "file_name_test",
		},
		{
			name:     "hyphen preserved in middle",
			input:    "file-name-test",
			expected: "file-name-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := FileName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileNameEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("single dash", func(t *testing.T) {
		t.Parallel()

		result := FileName("-")
		assert.Empty(t, result)
	})

	t.Run("single character", func(t *testing.T) {
		t.Parallel()

		result := FileName("a")
		assert.Equal(t, "a", result)
	})

	t.Run("only forbidden characters", func(t *testing.T) {
		t.Parallel()

		result := FileName("   ")
		assert.Equal(t, "_", result)
	})

	t.Run("multiple spaces collapse to single underscore", func(t *testing.T) {
		t.Parallel()

		result := FileName("a     b")
		assert.Equal(t, "a_b", result)
	})
}

func BenchmarkFileName(b *testing.B) {
	testCases := []string{
		"simple-file.txt",
		"Müller & Söhne (€100).txt",
		`":/\()?*
 {|}[¦]!#%<>~^'` + "`°§",
		"café-français-äöü-ÄÖÜ",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for range b.N {
				_ = FileName(tc)
			}
		})
	}
}
