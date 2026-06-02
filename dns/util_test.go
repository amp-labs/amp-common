package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordsEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		a, b      []Record
		ignoreTTL bool
		want      bool
	}{
		{
			name: "identical single record",
			a:    []Record{aRec("a.com.", "1.2.3.4")},
			b:    []Record{aRec("a.com.", "1.2.3.4")},
			want: true,
		},
		{
			name: "different lengths",
			a:    []Record{aRec("a.com.", "1.2.3.4")},
			b:    []Record{aRec("a.com.", "1.2.3.4"), aRec("a.com.", "5.6.7.8")},
			want: false,
		},
		{
			name: "same set different order",
			a:    []Record{aRec("a.com.", "1.2.3.4"), aRec("a.com.", "5.6.7.8")},
			b:    []Record{aRec("a.com.", "5.6.7.8"), aRec("a.com.", "1.2.3.4")},
			want: true,
		},
		{
			name: "different values",
			a:    []Record{aRec("a.com.", "1.2.3.4")},
			b:    []Record{aRec("a.com.", "9.9.9.9")},
			want: false,
		},
		{
			name:      "differing TTL not ignored",
			a:         []Record{{Type: TypeA, Name: "a.com.", Value: "1.2.3.4", TTL: 300}},
			b:         []Record{{Type: TypeA, Name: "a.com.", Value: "1.2.3.4", TTL: 60}},
			ignoreTTL: false,
			want:      false,
		},
		{
			name:      "differing TTL ignored",
			a:         []Record{{Type: TypeA, Name: "a.com.", Value: "1.2.3.4", TTL: 300}},
			b:         []Record{{Type: TypeA, Name: "a.com.", Value: "1.2.3.4", TTL: 60}},
			ignoreTTL: true,
			want:      true,
		},
		{
			name: "different multiplicities of same value",
			a:    []Record{aRec("a.com.", "1.2.3.4"), aRec("a.com.", "1.2.3.4")},
			b:    []Record{aRec("a.com.", "1.2.3.4"), aRec("a.com.", "5.6.7.8")},
			want: false,
		},
		{
			name: "both empty",
			a:    nil,
			b:    nil,
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, recordsEqual(tc.a, tc.b, tc.ignoreTTL))
			// Equality must be symmetric.
			assert.Equal(t, tc.want, recordsEqual(tc.b, tc.a, tc.ignoreTTL))
		})
	}
}
