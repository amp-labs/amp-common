package transport

import (
	"context"
	"strings"

	"github.com/amp-labs/amp-common/envutil"
)

// The environment variables that configure a transport. Each one can be set for a single provider
// by appending the provider's name, uppercased, to it (see providerKey), and that value wins over
// the variable without the suffix for transports created with WithProvider.
const (
	envPreferPooled          = "HTTP_TRANSPORT_PREFER_POOLED"
	envMaxIdleConns          = "HTTP_TRANSPORT_MAX_IDLE_CONNS"
	envMaxIdleConnsPerHost   = "HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST"
	envIdleConnTimeout       = "HTTP_TRANSPORT_IDLE_CONN_TIMEOUT"
	envTLSHandshakeTimeout   = "HTTP_TRANSPORT_TLS_HANDSHAKE_TIMEOUT"
	envExpectContinueTimeout = "HTTP_TRANSPORT_EXPECT_CONTINUE_TIMEOUT"
	envDisableHTTP2          = "HTTP_TRANSPORT_DISABLE_HTTP2"
	envForceAttemptHTTP2     = "HTTP_TRANSPORT_FORCE_ATTEMPT_HTTP2"
	envDialTimeout           = "HTTP_TRANSPORT_DIAL_TIMEOUT"
	envDialKeepAlive         = "HTTP_TRANSPORT_DIAL_KEEPALIVE"
)

// transportSettings are the variables create reads. A provider with any of them set for it gets a
// transport of its own (see providerTransportInstance). HTTP_TRANSPORT_PREFER_POOLED is not among
// them: it only chooses between the shared pooled and unpooled transports.
var transportSettings = []string{
	envMaxIdleConns,
	envMaxIdleConnsPerHost,
	envIdleConnTimeout,
	envTLSHandshakeTimeout,
	envExpectContinueTimeout,
	envDisableHTTP2,
	envForceAttemptHTTP2,
	envDialTimeout,
	envDialKeepAlive,
}

// readFunc is the shape of the envutil readers the settings use (envutil.Int, envutil.Bool,
// envutil.Duration).
type readFunc[T any] func(ctx context.Context, key string, opts ...envutil.Option[T]) envutil.Reader[T]

// setting returns the value of the variable key for provider: key with the provider's suffix when
// that is set, else key, else dflt. With no provider only key is read. A value that does not parse
// gives dflt, as it always has for these variables.
func setting[T any](ctx context.Context, read readFunc[T], key, provider string, dflt T) T { //nolint:ireturn
	if provider == "" {
		return read(ctx, key, envutil.Default(dflt)).ValueOrElse(dflt)
	}

	return read(ctx, providerKey(key, provider),
		envutil.Fallback(read(ctx, key)),
		envutil.Default(dflt)).
		ValueOrElse(dflt)
}

// providerKey returns the variable that sets key for one provider: key, an underscore, and the
// provider's name uppercased, with any character other than a letter or a digit replaced by an
// underscore. For example HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST for the provider "accuLynx" is
// HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_ACCULYNX.
func providerKey(key, provider string) string {
	return key + "_" + strings.Map(func(char rune) rune {
		switch {
		case char >= 'a' && char <= 'z':
			return char - 'a' + 'A'
		case char >= 'A' && char <= 'Z', char >= '0' && char <= '9':
			return char
		default:
			return '_'
		}
	}, provider)
}

// hasProviderSettings reports whether any of the transport settings is set for provider.
func hasProviderSettings(ctx context.Context, provider string) bool {
	for _, key := range transportSettings {
		if envutil.String(ctx, providerKey(key, provider)).HasValue() {
			return true
		}
	}

	return false
}
