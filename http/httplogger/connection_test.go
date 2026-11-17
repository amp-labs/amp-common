package httplogger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/http/httplogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loggedConnection logs one error line with ctx and returns its details' "connection" object, or nil.
func loggedConnection(t *testing.T, ctx context.Context) map[string]any {
	t.Helper()

	var logBuffer bytes.Buffer

	params := &httplogger.LogErrorParams{Logger: slog.New(slog.NewJSONHandler(&logBuffer, nil))}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com/path", nil)
	require.NoError(t, err)

	httplogger.LogError(ctx, req, errors.New("connection reset by peer"), //nolint:err113
		http.MethodGet, "corr-1", req.URL, params)

	var line struct {
		Details map[string]any `json:"details"`
	}

	require.NoError(t, json.Unmarshal(logBuffer.Bytes(), &line))

	conn, found := line.Details["connection"]
	if !found {
		return nil
	}

	fields, ok := conn.(map[string]any)
	require.True(t, ok, "connection is an object: %v", conn)

	return fields
}

func TestLogError_Connection(t *testing.T) {
	t.Parallel()

	remote := &net.TCPAddr{IP: net.IPv4(192, 0, 2, 10), Port: 443}
	local := &net.TCPAddr{IP: net.IPv4(10, 0, 0, 5), Port: 56826}

	tests := []struct {
		name string
		conn *httplogger.Connection
		want map[string]any
	}{
		{
			name: "no connection in the context",
			conn: nil,
			want: nil,
		},
		{
			name: "the transport reported nothing",
			conn: &httplogger.Connection{},
			want: nil,
		},
		{
			name: "failed before a connection was obtained",
			conn: &httplogger.Connection{Attempts: 1},
			want: map[string]any{"obtained": false},
		},
		{
			name: "a new connection",
			conn: &httplogger.Connection{Attempts: 1, Obtained: true, RemoteAddr: remote, LocalAddr: local},
			want: map[string]any{
				"obtained":   true,
				"reused":     false,
				"wasIdle":    false,
				"remoteAddr": "192.0.2.10:443",
				"localAddr":  "10.0.0.5:56826",
			},
		},
		{
			name: "a reused idle connection",
			conn: &httplogger.Connection{
				Attempts:   1,
				Obtained:   true,
				Reused:     true,
				WasIdle:    true,
				IdleTime:   95*time.Second + 250*time.Millisecond,
				RemoteAddr: remote,
				LocalAddr:  local,
			},
			want: map[string]any{
				"obtained":   true,
				"reused":     true,
				"wasIdle":    true,
				"idleMs":     float64(95250),
				"remoteAddr": "192.0.2.10:443",
				"localAddr":  "10.0.0.5:56826",
			},
		},
		{
			name: "sent again, and the second dial failed",
			conn: &httplogger.Connection{Attempts: 2},
			want: map[string]any{"obtained": false, "attempts": float64(2)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if tt.conn != nil {
				ctx = httplogger.WithConnection(ctx, tt.conn)
			}

			assert.Equal(t, tt.want, loggedConnection(t, ctx))
		})
	}
}

func TestLogResponse_Connection(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	params := &httplogger.LogResponseParams{Logger: slog.New(slog.NewJSONHandler(&logBuffer, nil))}

	ctx := httplogger.WithConnection(t.Context(), &httplogger.Connection{
		Attempts:   1,
		Obtained:   true,
		Reused:     true,
		RemoteAddr: &net.TCPAddr{IP: net.IPv4(192, 0, 2, 10), Port: 443},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com/users", nil)
	require.NoError(t, err)

	resp := &http.Response{Status: "200 OK", StatusCode: http.StatusOK, Request: req, Header: make(http.Header)}

	httplogger.LogResponse(ctx, resp, nil, http.MethodGet, "corr-1", req.URL, params)

	var line struct {
		Details struct {
			Connection map[string]any `json:"connection"`
		} `json:"details"`
	}

	require.NoError(t, json.Unmarshal(logBuffer.Bytes(), &line))
	assert.Equal(t, map[string]any{
		"obtained":   true,
		"reused":     true,
		"wasIdle":    false,
		"remoteAddr": "192.0.2.10:443",
	}, line.Details.Connection)
}
