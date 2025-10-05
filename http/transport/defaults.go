package transport

import "time"

const (
	// defaultIdleConnTimeout is the maximum amount of time an idle connection will remain idle before closing.
	defaultIdleConnTimeout = 90 * time.Second

	// defaultMaxIdleConns controls the maximum number of idle (keep-alive) connections across all hosts.
	defaultMaxIdleConns = 100

	// defaultTLSHandshakeTimeout specifies the maximum amount of time waiting to complete a TLS handshake.
	defaultTLSHandshakeTimeout = 10 * time.Second

	// defaultExpectContinueTimeout specifies the amount of time to wait for a server's first response headers
	// after fully writing the request headers if the request has an "Expect: 100-continue" header.
	defaultExpectContinueTimeout = 1 * time.Second

	// defaultForceAttemptHTTP2 controls whether HTTP/2 is enabled when a non-zero Dial, DialTLS, or DialContext
	// function or TLSClientConfig is provided.
	defaultForceAttemptHTTP2 = false

	// defaultTransportDialTimeout is the maximum amount of time a dial will wait for a connection to complete.
	defaultTransportDialTimeout = 30 * time.Second //nolint:gomnd,mnd

	// defaultKeepAlive specifies the interval between keep-alive probes for an active network connection.
	defaultKeepAlive = 30 * time.Second //nolint:gomnd,mnd
)
