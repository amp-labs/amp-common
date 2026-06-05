package dns

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIPToRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ip       net.IP
		wantType RecordType
		wantOK   bool
	}{
		{
			// net.ParseIP returns IPv4 addresses in 16-byte IPv4-mapped form;
			// classification must still report them as A records.
			name:     "parsed IPv4 (16-byte mapped form)",
			ip:       net.ParseIP("1.2.3.4"),
			wantType: TypeA,
			wantOK:   true,
		},
		{
			name:     "4-byte IPv4",
			ip:       net.IP{1, 2, 3, 4},
			wantType: TypeA,
			wantOK:   true,
		},
		{
			name:     "explicit IPv4-mapped IPv6",
			ip:       net.ParseIP("::ffff:1.2.3.4"),
			wantType: TypeA,
			wantOK:   true,
		},
		{
			name:     "IPv6",
			ip:       net.ParseIP("2001:db8::1"),
			wantType: TypeAAAA,
			wantOK:   true,
		},
		{
			name:   "nil IP",
			ip:     nil,
			wantOK: false,
		},
		{
			name:   "malformed length",
			ip:     net.IP{1, 2, 3},
			wantOK: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			record, ok := ipToRecord(testCase.ip)

			require.Equal(t, testCase.wantOK, ok)

			if !testCase.wantOK {
				assert.Equal(t, Record{}, record)

				return
			}

			assert.Equal(t, testCase.wantType, record.Type)
			// The synthetic record carries the IP itself as both name and value.
			assert.Equal(t, testCase.ip.String(), record.Name)
			assert.Equal(t, testCase.ip.String(), record.Value)
		})
	}
}

func TestParseHostAndPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addr     string
		wantHost string
		wantPort uint16
		wantErr  bool
	}{
		{name: "hostname with port", addr: "example.com:443", wantHost: "example.com", wantPort: 443},
		{name: "IPv4 with port", addr: "1.2.3.4:80", wantHost: "1.2.3.4", wantPort: 80},
		{name: "bracketed IPv6 with port", addr: "[::1]:8080", wantHost: "::1", wantPort: 8080},
		{name: "max port", addr: "example.com:65535", wantHost: "example.com", wantPort: 65535},
		{name: "port zero", addr: "example.com:0", wantHost: "example.com", wantPort: 0},
		{name: "missing port", addr: "example.com", wantErr: true},
		{name: "service name instead of numeric port", addr: "example.com:http", wantErr: true},
		{name: "port out of range", addr: "example.com:65536", wantErr: true},
		{name: "negative port", addr: "example.com:-1", wantErr: true},
		{name: "empty address", addr: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			host, port, err := parseHostAndPort(tc.addr)

			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantHost, host)
			assert.Equal(t, tc.wantPort, port)
		})
	}
}

func TestFilterIPFunctions(t *testing.T) {
	t.Parallel()

	var (
		v4a    = net.ParseIP("1.2.3.4")
		v4b    = net.ParseIP("5.6.7.8")
		v6a    = net.ParseIP("2001:db8::1")
		v6b    = net.ParseIP("::1")
		mapped = net.ParseIP("::ffff:9.9.9.9") // IPv4-mapped: dialable as v4, not as v6
		bad    = net.IP{1, 2, 3}               // malformed, dropped by every filter
	)

	// Interleave families so the tests catch ordering behavior, not just membership.
	mixed := []net.IP{v6a, v4a, bad, v6b, v4b, mapped}

	t.Run("filterIPv4 keeps IPv4 and mapped addresses in order", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []net.IP{v4a, v4b, mapped}, filterIPv4(mixed))
	})

	t.Run("filterIPv6 keeps only genuine IPv6 addresses", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []net.IP{v6a, v6b}, filterIPv6(mixed))
	})

	t.Run("filterAnyIP orders IPv4 before IPv6 and drops malformed", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []net.IP{v4a, v4b, mapped, v6a, v6b}, filterAnyIP(mixed))
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, filterIPv4(nil))
		assert.Empty(t, filterIPv6(nil))
		assert.Empty(t, filterAnyIP(nil))
	})
}
