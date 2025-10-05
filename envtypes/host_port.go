// Package envtypes provides common types used for parsing environment variables.
package envtypes

import (
	"strconv"

	"github.com/amp-labs/amp-common/tuple"
)

// TupleToHostPort converts a tuple of host and port to a HostPort struct.
func TupleToHostPort(t tuple.Tuple2[string, uint16]) HostPort {
	return HostPort{
		Host: t.First(),
		Port: t.Second(),
	}
}

// HostPort represents a network host and port combination.
// It is commonly used for parsing network addresses from environment variables.
type HostPort struct {
	Host string
	Port uint16
}

// String returns the host:port representation as a string.
func (hp HostPort) String() string {
	return hp.Host + ":" + strconv.FormatUint(uint64(hp.Port), 10)
}

// AsTuple converts the HostPort to a Tuple2 of host and port.
func (hp HostPort) AsTuple() tuple.Tuple2[string, uint16] {
	return tuple.NewTuple2[string, uint16](hp.Host, hp.Port)
}
