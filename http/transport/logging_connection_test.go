package transport

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"sync"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/http/httplogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newConnectionTestClient returns a client whose logging transport wraps its own *http.Transport, so the
// connection pool is the test's alone, and the logger its lines go to.
func newConnectionTestClient(t *testing.T) (*http.Client, *captureLogger) {
	t.Helper()

	captureLog := newCaptureLogger()
	log := slog.New(captureLog.handler())

	base := &http.Transport{}
	t.Cleanup(base.CloseIdleConnections)

	trans := NewLoggingTransport(t.Context(), base,
		&httplogger.LogRequestParams{Logger: log},
		&httplogger.LogResponseParams{Logger: log},
		&httplogger.LogErrorParams{Logger: log})

	return &http.Client{Transport: trans}, captureLog
}

// get sends a GET to url and reads and closes the body, so the connection goes back to the pool.
func get(t *testing.T, client *http.Client, url string) error {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	return nil
}

// loggedConnection returns the "connection" object of the line with the given message, the nth such line
// (from 0), or nil when that line has none.
func loggedConnection(t *testing.T, captureLog *captureLogger, message string, nth int) map[string]any {
	t.Helper()

	seen := 0

	for _, entry := range captureLog.logs {
		if entry.message != message {
			continue
		}

		if seen < nth {
			seen++

			continue
		}

		details, ok := entry.attrs["details"].(map[string]any)
		require.True(t, ok)

		conn, found := details["connection"]
		if !found {
			return nil
		}

		fields, ok := conn.(map[string]any)
		require.True(t, ok)

		return fields
	}

	require.Failf(t, "line not logged", "%q #%d", message, nth)

	return nil
}

func TestLoggingTransport_Connection_NewThenIdleReuse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(server.Close)

	client, captureLog := newConnectionTestClient(t)

	require.NoError(t, get(t, client, server.URL))

	const idle = 20 * time.Millisecond

	time.Sleep(idle)

	require.NoError(t, get(t, client, server.URL))

	first := loggedConnection(t, captureLog, httplogger.DefaultLogResponseMessage, 0)
	assert.Equal(t, true, first["obtained"])
	assert.Equal(t, false, first["reused"])
	assert.Equal(t, false, first["wasIdle"])
	assert.NotContains(t, first, "idleMs")
	assert.NotContains(t, first, "attempts")
	assert.Equal(t, server.Listener.Addr().String(), first["remoteAddr"])
	assert.NotEmpty(t, first["localAddr"])

	second := loggedConnection(t, captureLog, httplogger.DefaultLogResponseMessage, 1)
	assert.Equal(t, true, second["obtained"])
	assert.Equal(t, true, second["reused"])
	assert.Equal(t, true, second["wasIdle"])
	assert.GreaterOrEqual(t, second["idleMs"], idle.Milliseconds())
	assert.Equal(t, server.Listener.Addr().String(), second["remoteAddr"])
	assert.Equal(t, first["localAddr"], second["localAddr"], "the same TCP connection")
}

func TestLoggingTransport_Connection_DialFailure(t *testing.T) {
	t.Parallel()

	// An address nothing listens on: a listener's, closed.
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	client, captureLog := newConnectionTestClient(t)

	require.Error(t, get(t, client, "http://"+addr+"/"))

	assert.Equal(t, map[string]any{"obtained": false},
		loggedConnection(t, captureLog, httplogger.DefaultLogErrorMessage, 0))
}

// net/http sends a replayable request again on a new connection when a reused one fails before any
// response arrives; the line says so with attempts, and describes the connection that answered.
func TestLoggingTransport_Connection_SentAgainAfterReusedConnectionFails(t *testing.T) {
	t.Parallel()

	var (
		mu       sync.Mutex
		requests = map[string]int{}
	)

	// The second request on any connection gets the connection closed under it.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests[r.RemoteAddr]++
		count := requests[r.RemoteAddr]
		mu.Unlock()

		if count < 2 {
			_, _ = io.WriteString(w, "ok")

			return
		}

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			panic("no hijacker")
		}

		conn, _, err := hijacker.Hijack()
		if err != nil {
			panic(err)
		}

		_ = conn.Close()
	}))
	t.Cleanup(server.Close)

	client, captureLog := newConnectionTestClient(t)

	require.NoError(t, get(t, client, server.URL))
	require.NoError(t, get(t, client, server.URL))

	first := loggedConnection(t, captureLog, httplogger.DefaultLogResponseMessage, 0)
	second := loggedConnection(t, captureLog, httplogger.DefaultLogResponseMessage, 1)

	assert.Equal(t, 2, second["attempts"])
	assert.Equal(t, true, second["obtained"])
	assert.Equal(t, false, second["reused"])
	assert.NotEqual(t, first["localAddr"], second["localAddr"], "a new TCP connection")
}

func TestLoggingTransport_Connection_ComposesWithAnExistingTrace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(server.Close)

	client, captureLog := newConnectionTestClient(t)

	var callerSaw httptrace.GotConnInfo

	ctx := httptrace.WithClientTrace(t.Context(), &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { callerSaw = info },
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.NotNil(t, callerSaw.Conn, "the caller's trace still sees GotConn")

	conn := loggedConnection(t, captureLog, httplogger.DefaultLogResponseMessage, 0)
	assert.Equal(t, callerSaw.Conn.LocalAddr().String(), conn["localAddr"])
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestLoggingTransport_Connection_TransportThatReportsNothing(t *testing.T) {
	t.Parallel()

	type key struct{}

	captureLog := newCaptureLogger()
	log := slog.New(captureLog.handler())

	var sent *http.Request

	trans := NewLoggingTransport(t.Context(), roundTripFunc(func(req *http.Request) (*http.Response, error) {
		sent = req

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{},
			Body:       http.NoBody,
		}, nil
	}),
		&httplogger.LogRequestParams{Logger: log},
		&httplogger.LogResponseParams{Logger: log},
		&httplogger.LogErrorParams{Logger: log})

	req, err := http.NewRequestWithContext(context.WithValue(t.Context(), key{}, "kept"),
		http.MethodGet, "https://api.example.com/test", nil)
	require.NoError(t, err)

	resp, err := trans.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.NotNil(t, sent)
	assert.Equal(t, "kept", sent.Context().Value(key{}), "the wrapped transport sees the caller's context")
	assert.NotNil(t, httptrace.ContextClientTrace(sent.Context()))
	assert.Nil(t, httptrace.ContextClientTrace(req.Context()), "the caller's request is not changed")

	assert.Nil(t, loggedConnection(t, captureLog, httplogger.DefaultLogResponseMessage, 0))
}
