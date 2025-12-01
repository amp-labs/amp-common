package envutil

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
)

type envContextKey string

func WithEnvOverride(ctx context.Context, key string, value string) context.Context {
	return contexts.WithValue[envContextKey, string](ctx, envContextKey(key), value)
}

func getEnvOverride(ctx context.Context, key string) (string, bool) {
	return contexts.GetValue[envContextKey, string](ctx, envContextKey(key))
}
