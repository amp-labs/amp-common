package feature

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/amp-labs/amp-common/logger"
)

// maxBodyBytes bounds the request bodies the Handler accepts.
const maxBodyBytes = 1 << 20

// errBadRequest is wrapped by errors returned for malformed request bodies.
var errBadRequest = errors.New("bad request")

// Handler returns an http.Handler that exposes the registry for inspection
// and runtime changes. Routes are relative to the handler's root, so mount it
// with http.StripPrefix:
//
//	mux.Handle("/internal/flags/", http.StripPrefix("/internal/flags", feature.Handler(reg)))
//
// Routes:
//
//	GET    /                 list every flag as Info
//	GET    /{name}           one flag as Info
//	PUT    /{name}           set the rollout; the body is a Rollout, as an object or a shorthand string
//	DELETE /{name}           reset the flag to its code default
//	POST   /{name}/evaluate  evaluate the flag for the Subject in the body (may be empty); returns Result
//
// Errors are returned as {"error": "..."} with a 400 or 404 status.
//
// The handler does no authentication. Mount it on an internal listener or
// behind your own middleware. A nil registry means Default().
func Handler(reg *Registry) http.Handler {
	if reg == nil {
		reg = Default()
	}

	hnd := &handler{registry: reg}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", hnd.list)
	mux.HandleFunc("GET /{name}", hnd.get)
	mux.HandleFunc("PUT /{name}", hnd.set)
	mux.HandleFunc("DELETE /{name}", hnd.reset)
	mux.HandleFunc("POST /{name}/evaluate", hnd.evaluate)

	return mux
}

type handler struct {
	registry *Registry
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *handler) list(writer http.ResponseWriter, req *http.Request) {
	writeJSON(req.Context(), writer, http.StatusOK, h.registry.Infos())
}

func (h *handler) get(writer http.ResponseWriter, req *http.Request) {
	def, ok := h.lookup(writer, req)
	if !ok {
		return
	}

	writeJSON(req.Context(), writer, http.StatusOK, def.Info())
}

func (h *handler) set(writer http.ResponseWriter, req *http.Request) {
	def, ok := h.lookup(writer, req)
	if !ok {
		return
	}

	var rollout Rollout

	err := readJSON(writer, req, &rollout)
	if err != nil {
		writeError(req.Context(), writer, http.StatusBadRequest, err)

		return
	}

	err = def.Set(rollout)
	if err != nil {
		writeError(req.Context(), writer, http.StatusBadRequest, err)

		return
	}

	writeJSON(req.Context(), writer, http.StatusOK, def.Info())
}

func (h *handler) reset(writer http.ResponseWriter, req *http.Request) {
	def, ok := h.lookup(writer, req)
	if !ok {
		return
	}

	def.Reset()

	writeJSON(req.Context(), writer, http.StatusOK, def.Info())
}

func (h *handler) evaluate(writer http.ResponseWriter, req *http.Request) {
	def, ok := h.lookup(writer, req)
	if !ok {
		return
	}

	var subject Subject

	err := readJSON(writer, req, &subject)
	if err != nil && !errors.Is(err, io.EOF) {
		writeError(req.Context(), writer, http.StatusBadRequest, err)

		return
	}

	writeJSON(req.Context(), writer, http.StatusOK, def.Evaluate(req.Context(), subject))
}

// lookup resolves the {name} path value, writing a 404 if it is not defined.
func (h *handler) lookup(writer http.ResponseWriter, req *http.Request) (*Flag, bool) {
	name := Name(req.PathValue("name"))

	def, ok := h.registry.Lookup(name)
	if !ok {
		writeError(req.Context(), writer, http.StatusNotFound, fmt.Errorf("%writer: %q", ErrUndefined, name))

		return nil, false
	}

	return def, true
}

// readJSON decodes the request body into target. An empty body yields io.EOF.
func readJSON(writer http.ResponseWriter, req *http.Request, target any) error {
	body := http.MaxBytesReader(writer, req.Body, maxBodyBytes)

	err := json.NewDecoder(body).Decode(target)
	if errors.Is(err, io.EOF) {
		return err
	}

	if err != nil {
		return fmt.Errorf("%writer: %writer", errBadRequest, err)
	}

	return nil
}

func writeError(ctx context.Context, writer http.ResponseWriter, status int, err error) {
	writeJSON(ctx, writer, status, errorResponse{Error: err.Error()})
}

func writeJSON(ctx context.Context, writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	err := json.NewEncoder(writer).Encode(value)
	if err != nil {
		logger.Get(ctx).Warn("failed to write feature flag response", "error", err)
	}
}
