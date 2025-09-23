package transport

import "time"

const (
	defaultIdleConnTimeout       = 90 * time.Second
	defaultMaxIdleConns          = 100
	defaultTLSHandshakeTimeout   = 10 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
	defaultForceAttemptHTTP2     = false
	defaultTransportDialTimeout  = 30 * time.Second //nolint:gomnd,mnd
	defaultKeepAlive             = 30 * time.Second //nolint:gomnd,mnd
)
