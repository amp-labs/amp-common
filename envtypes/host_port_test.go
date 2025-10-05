package envtypes

import (
	"testing"

	"github.com/amp-labs/amp-common/tuple"
	"github.com/stretchr/testify/assert"
)

func TestTupleToHostPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    tuple.Tuple2[string, uint16]
		expected HostPort
	}{
		{
			name:  "standard host and port",
			input: tuple.NewTuple2[string, uint16]("localhost", 8080),
			expected: HostPort{
				Host: "localhost",
				Port: 8080,
			},
		},
		{
			name:  "IP address and port",
			input: tuple.NewTuple2[string, uint16]("192.168.1.1", 443),
			expected: HostPort{
				Host: "192.168.1.1",
				Port: 443,
			},
		},
		{
			name:  "empty host",
			input: tuple.NewTuple2[string, uint16]("", 3000),
			expected: HostPort{
				Host: "",
				Port: 3000,
			},
		},
		{
			name:  "zero port",
			input: tuple.NewTuple2[string, uint16]("example.com", 0),
			expected: HostPort{
				Host: "example.com",
				Port: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := TupleToHostPort(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHostPort_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hostPort HostPort
		expected string
	}{
		{
			name: "standard host and port",
			hostPort: HostPort{
				Host: "localhost",
				Port: 8080,
			},
			expected: "localhost:8080",
		},
		{
			name: "IP address and port",
			hostPort: HostPort{
				Host: "192.168.1.1",
				Port: 443,
			},
			expected: "192.168.1.1:443",
		},
		{
			name: "domain with high port",
			hostPort: HostPort{
				Host: "example.com",
				Port: 65535,
			},
			expected: "example.com:65535",
		},
		{
			name: "empty host",
			hostPort: HostPort{
				Host: "",
				Port: 3000,
			},
			expected: ":3000",
		},
		{
			name: "zero port",
			hostPort: HostPort{
				Host: "example.com",
				Port: 0,
			},
			expected: "example.com:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.hostPort.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHostPort_AsTuple(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hostPort HostPort
	}{
		{
			name: "standard host and port",
			hostPort: HostPort{
				Host: "localhost",
				Port: 8080,
			},
		},
		{
			name: "IP address and port",
			hostPort: HostPort{
				Host: "10.0.0.1",
				Port: 22,
			},
		},
		{
			name: "empty host",
			hostPort: HostPort{
				Host: "",
				Port: 5432,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.hostPort.AsTuple()
			assert.Equal(t, tt.hostPort.Host, result.First())
			assert.Equal(t, tt.hostPort.Port, result.Second())
		})
	}
}

func TestHostPort_RoundTrip(t *testing.T) {
	t.Parallel()

	original := HostPort{
		Host: "example.com",
		Port: 9000,
	}

	// Convert to tuple and back
	asTuple := original.AsTuple()
	backToHostPort := TupleToHostPort(asTuple)

	assert.Equal(t, original, backToHostPort)
}
