package envtypes

import (
	"strconv"

	"github.com/amp-labs/amp-common/tuple"
)

func TupleToHostPort(t tuple.Tuple2[string, uint16]) HostPort {
	return HostPort{
		Host: t.First(),
		Port: t.Second(),
	}
}

type HostPort struct {
	Host string
	Port uint16
}

func (hp HostPort) String() string {
	return hp.Host + ":" + strconv.FormatUint(uint64(hp.Port), 10)
}

func (hp HostPort) AsTuple() tuple.Tuple2[string, uint16] {
	return tuple.NewTuple2[string, uint16](hp.Host, hp.Port)
}
