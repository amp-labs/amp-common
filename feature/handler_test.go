package feature

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, hnd http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequestWithContext(t.Context(), method, path, reader)
	rec := httptest.NewRecorder()
	hnd.ServeHTTP(rec, req)

	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var out T

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())

	return out
}

func TestHandlerList(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Define("b", Off(), Description("bee"))
	reg.Define("a", On(), Owner("team"))

	rec := serve(t, Handler(reg), http.MethodGet, "/", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	infos := decode[[]Info](t, rec)
	require.Len(t, infos, 2)
	assert.Equal(t, Name("a"), infos[0].Name)
	assert.Equal(t, "team", infos[0].Owner)
	assert.Equal(t, Name("b"), infos[1].Name)
	assert.Equal(t, "bee", infos[1].Description)
}

func TestHandlerGet(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Define("a", On())
	hnd := Handler(reg)

	rec := serve(t, hnd, http.MethodGet, "/a", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, decode[Info](t, rec).Current.Equal(On()))

	rec = serve(t, hnd, http.MethodGet, "/missing", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, decode[errorResponse](t, rec).Error, "undefined")
}

func TestHandlerSet(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("a", Off())
	hnd := Handler(reg)
	ctx := t.Context()

	rec := serve(t, hnd, http.MethodPut, "/a", `{"default": true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, decode[Info](t, rec).Overridden)
	assert.True(t, flag.Enabled(ctx))

	rec = serve(t, hnd, http.MethodPut, "/a", `"25% user"`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, flag.Current().Equal(Off().Percent(dimUser, 25)))

	rec = serve(t, hnd, http.MethodPut, "/a", `{"percentages": [{"dimension": "user", "percent": 500}]}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, decode[errorResponse](t, rec).Error, "invalid rollout")
	assert.True(t, flag.Current().Equal(Off().Percent(dimUser, 25)), "invalid request leaves the flag alone")

	rec = serve(t, hnd, http.MethodPut, "/a", `not json`)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, hnd, http.MethodPut, "/a", "")
	require.Equal(t, http.StatusBadRequest, rec.Code, "an empty body is not a rollout")

	rec = serve(t, hnd, http.MethodPut, "/missing", `{"default": true}`)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandlerReset(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("a", Off())
	hnd := Handler(reg)

	require.NoError(t, flag.Set(On()))

	rec := serve(t, hnd, http.MethodDelete, "/a", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, decode[Info](t, rec).Overridden)
	assert.False(t, flag.Overridden())

	rec = serve(t, hnd, http.MethodDelete, "/missing", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandlerEvaluate(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Define("e", Off().Allow(dimUser, "vip"))
	hnd := Handler(reg)

	rec := serve(t, hnd, http.MethodPost, "/e/evaluate", `{"user": "vip"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	result := decode[Result](t, rec)
	assert.True(t, result.Enabled)
	assert.Equal(t, ReasonAllowed, result.Reason)

	rec = serve(t, hnd, http.MethodPost, "/e/evaluate", "")
	require.Equal(t, http.StatusOK, rec.Code, "an empty body is an empty subject")

	result = decode[Result](t, rec)
	assert.False(t, result.Enabled)
	assert.Equal(t, ReasonDefault, result.Reason)

	rec = serve(t, hnd, http.MethodPost, "/e/evaluate", `[1, 2]`)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, hnd, http.MethodPost, "/missing/evaluate", `{}`)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandlerMountedWithStripPrefix(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Define("a", On())

	mux := http.NewServeMux()
	mux.Handle("/internal/flags/", http.StripPrefix("/internal/flags", Handler(reg)))

	rec := serve(t, mux, http.MethodGet, "/internal/flags/", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, decode[[]Info](t, rec), 1)

	rec = serve(t, mux, http.MethodGet, "/internal/flags/a", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, Name("a"), decode[Info](t, rec).Name)
}

func TestHandlerNilRegistryUsesDefault(t *testing.T) {
	t.Parallel()

	Define("feature-test-handler-default", Off())

	rec := serve(t, Handler(nil), http.MethodGet, "/feature-test-handler-default", "")
	require.Equal(t, http.StatusOK, rec.Code)
}
