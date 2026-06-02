package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rt   RecordType
		want string
	}{
		{TypeA, "A"},
		{TypeAAAA, "AAAA"},
		{TypeCNAME, "CNAME"},
		{TypeMX, "MX"},
		{TypeNS, "NS"},
		{TypeTXT, "TXT"},
		{TypeSOA, "SOA"},
		{TypePTR, "PTR"},
		{TypeSRV, "SRV"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, tc.rt.String())
		})
	}
}

func TestRecord_String(t *testing.T) {
	t.Parallel()

	r := Record{Type: TypeA, Name: "example.com.", Value: "1.2.3.4", TTL: 300}
	assert.Equal(t, "example.com. A: 1.2.3.4 (TTL: 300)", r.String())
}
