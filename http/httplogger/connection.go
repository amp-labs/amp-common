package httplogger

import (
	"context"
	"net"
	"time"

	"github.com/amp-labs/amp-common/contexts"
)

const (
	// contextKeyConnection is the context key for the Connection a request went out on.
	contextKeyConnection contextKey = "connection"

	// Field names of the "connection" object LogResponse and LogError add to their details.
	fieldConnection = "connection"
	fieldObtained   = "obtained"
	fieldReused     = "reused"
	fieldWasIdle    = "wasIdle"
	fieldIdleMs     = "idleMs"
	fieldRemoteAddr = "remoteAddr"
	fieldLocalAddr  = "localAddr"
	fieldAttempts   = "attempts"
)

// Connection describes the connection an HTTP request went out on, as net/http/httptrace reports it
// (GetConn and GotConn). The logging transport in http/transport records one per request and puts it in
// the context it logs with (WithConnection), so LogResponse and LogError add it to their line as a
// "connection" object. That says whether a failed request had been sent on a pooled connection, how long
// that connection had sat idle, and which address it was connected to.
//
// A Connection is filled in by the RoundTrip that owns it and read only after that RoundTrip returns.
type Connection struct {
	// Attempts is how many times the transport asked for a connection for this request (httptrace's
	// GetConn). It is more than 1 when net/http sent the request again on another connection, which it
	// does for a replayable request after a reused connection failed before any response arrived. The
	// other fields describe the last attempt. Zero means the transport reported nothing, and nothing is
	// logged.
	Attempts int

	// Obtained is false when the last attempt failed before it had a connection: a DNS, dial, proxy or TLS
	// failure, or a cancellation while waiting for one.
	Obtained bool

	// Reused is whether the connection had carried an earlier request.
	Reused bool

	// WasIdle is whether the connection was taken from the idle pool.
	WasIdle bool

	// IdleTime is how long the connection had sat in the idle pool, when WasIdle.
	IdleTime time.Duration

	// RemoteAddr is the address the connection was connected to (a proxy's, when one is used).
	RemoteAddr net.Addr

	// LocalAddr is the connection's local address. With RemoteAddr it names the TCP connection, so the
	// lines of every request sent on one connection can be found, and the connection's age read from the
	// first of them.
	LocalAddr net.Addr
}

// WithConnection returns a context carrying conn, which LogResponse and LogError called with that context
// add to their line as a "connection" object.
func WithConnection(ctx context.Context, conn *Connection) context.Context {
	return contexts.WithValue[contextKey, *Connection](ctx, contextKeyConnection, conn)
}

// connectionFields returns the "connection" object for the Connection in ctx, or nil when there is none
// or the transport reported nothing.
func connectionFields(ctx context.Context) map[string]any {
	conn, found := contexts.GetValue[contextKey, *Connection](ctx, contextKeyConnection)
	if !found || conn == nil || conn.Attempts == 0 {
		return nil
	}

	fields := map[string]any{fieldObtained: conn.Obtained}

	if conn.Attempts > 1 {
		fields[fieldAttempts] = conn.Attempts
	}

	if !conn.Obtained {
		return fields
	}

	fields[fieldReused] = conn.Reused
	fields[fieldWasIdle] = conn.WasIdle

	if conn.WasIdle {
		fields[fieldIdleMs] = conn.IdleTime.Milliseconds()
	}

	if conn.RemoteAddr != nil {
		fields[fieldRemoteAddr] = conn.RemoteAddr.String()
	}

	if conn.LocalAddr != nil {
		fields[fieldLocalAddr] = conn.LocalAddr.String()
	}

	return fields
}
