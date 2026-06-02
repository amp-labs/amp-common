package utils //nolint:revive // utils is an appropriate package name for utility functions

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPrivateIPString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantPrivate bool
		wantValid   bool
	}{
		{name: "IPv4 loopback", input: "127.0.0.1", wantPrivate: true, wantValid: true},
		{name: "RFC1918 10.x", input: "10.1.2.3", wantPrivate: true, wantValid: true},
		{name: "RFC1918 172.16.x", input: "172.16.5.4", wantPrivate: true, wantValid: true},
		{name: "RFC1918 192.168.x", input: "192.168.0.1", wantPrivate: true, wantValid: true},
		{name: "link-local 169.254.x", input: "169.254.1.1", wantPrivate: true, wantValid: true},
		{name: "IPv6 loopback", input: "::1", wantPrivate: true, wantValid: true},
		{name: "IPv6 link-local", input: "fe80::1", wantPrivate: true, wantValid: true},
		{name: "IPv6 unique local", input: "fc00::1", wantPrivate: true, wantValid: true},
		{name: "public IPv4", input: "8.8.8.8", wantPrivate: false, wantValid: true},
		{name: "public IPv4 just outside 172.16/12", input: "172.32.0.1", wantPrivate: false, wantValid: true},
		{name: "public IPv6", input: "2001:4860:4860::8888", wantPrivate: false, wantValid: true},
		{name: "empty string", input: "", wantPrivate: false, wantValid: false},
		{name: "not an IP", input: "not-an-ip", wantPrivate: false, wantValid: false},
		{name: "hostname", input: "example.com", wantPrivate: false, wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			private, valid := IsPrivateIPString(tt.input)
			assert.Equal(t, tt.wantValid, valid, "valid mismatch")
			assert.Equal(t, tt.wantPrivate, private, "private mismatch")
		})
	}
}

func TestIsPublicIPString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantPublic bool
		wantValid  bool
	}{
		{name: "public IPv4", input: "8.8.8.8", wantPublic: true, wantValid: true},
		{name: "public IPv6", input: "2001:4860:4860::8888", wantPublic: true, wantValid: true},
		{name: "IPv4 loopback", input: "127.0.0.1", wantPublic: false, wantValid: true},
		{name: "RFC1918 10.x", input: "10.0.0.1", wantPublic: false, wantValid: true},
		{name: "RFC1918 192.168.x", input: "192.168.1.1", wantPublic: false, wantValid: true},
		{name: "link-local 169.254.x", input: "169.254.0.1", wantPublic: false, wantValid: true},
		{name: "IPv6 loopback", input: "::1", wantPublic: false, wantValid: true},
		{name: "IPv6 link-local", input: "fe80::1", wantPublic: false, wantValid: true},
		{name: "empty string", input: "", wantPublic: false, wantValid: false},
		{name: "not an IP", input: "garbage", wantPublic: false, wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			public, valid := IsPublicIPString(tt.input)
			assert.Equal(t, tt.wantValid, valid, "valid mismatch")
			assert.Equal(t, tt.wantPublic, public, "public mismatch")
		})
	}
}

func TestIsPrivateIPAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input net.IP
		want  bool
	}{
		{name: "nil IP", input: nil, want: false},
		{name: "IPv4 loopback", input: net.ParseIP("127.0.0.1"), want: true},
		{name: "RFC1918 10.x", input: net.ParseIP("10.0.0.1"), want: true},
		{name: "RFC1918 172.16.x", input: net.ParseIP("172.20.0.1"), want: true},
		{name: "RFC1918 192.168.x", input: net.ParseIP("192.168.100.100"), want: true},
		{name: "link-local 169.254.x", input: net.ParseIP("169.254.10.10"), want: true},
		{name: "IPv6 loopback", input: net.ParseIP("::1"), want: true},
		{name: "IPv6 link-local", input: net.ParseIP("fe80::abcd"), want: true},
		{name: "IPv6 unique local", input: net.ParseIP("fd00::1"), want: true},
		{name: "public IPv4", input: net.ParseIP("1.1.1.1"), want: false},
		{name: "public IPv6", input: net.ParseIP("2606:4700:4700::1111"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsPrivateIPAddress(tt.input))
		})
	}
}

func TestIsPublicIPAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input net.IP
		want  bool
	}{
		{name: "nil IP", input: nil, want: false},
		{name: "public IPv4", input: net.ParseIP("1.1.1.1"), want: true},
		{name: "public IPv6", input: net.ParseIP("2606:4700:4700::1111"), want: true},
		{name: "IPv4 loopback", input: net.ParseIP("127.0.0.1"), want: false},
		{name: "RFC1918 10.x", input: net.ParseIP("10.255.255.255"), want: false},
		{name: "RFC1918 192.168.x", input: net.ParseIP("192.168.0.1"), want: false},
		{name: "link-local 169.254.x", input: net.ParseIP("169.254.0.1"), want: false},
		{name: "IPv6 loopback", input: net.ParseIP("::1"), want: false},
		{name: "IPv6 link-local", input: net.ParseIP("fe80::1"), want: false},
		{name: "IPv6 unique local", input: net.ParseIP("fc00::1"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsPublicIPAddress(tt.input))
		})
	}
}

// TestPublicPrivateAreComplementary verifies that for any valid IP, exactly one
// of IsPublicIPAddress / IsPrivateIPAddress is true.
func TestPublicPrivateAreComplementary(t *testing.T) {
	t.Parallel()

	ips := []string{
		"8.8.8.8",
		"1.1.1.1",
		"127.0.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"169.254.1.1",
		"::1",
		"fe80::1",
		"fc00::1",
		"2001:4860:4860::8888",
	}

	for _, s := range ips {
		t.Run(s, func(t *testing.T) {
			t.Parallel()

			ip := net.ParseIP(s)
			require.NotNil(t, ip)

			assert.NotEqual(t, IsPublicIPAddress(ip), IsPrivateIPAddress(ip),
				"an IP must be exactly one of public or private")
		})
	}
}
