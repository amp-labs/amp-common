// Package printable provides utilities for converting HTTP request/response bodies into
// human-readable or loggable formats.
//
// This package handles the complexity of converting arbitrary HTTP bodies into formats
// suitable for logging and debugging. It automatically detects binary content, handles
// various character encodings, and can base64-encode non-printable content.
//
// # Basic Usage
//
//	import "github.com/amp-labs/amp-common/http/printable"
//
//	// Convert request body to printable format
//	payload, err := printable.Request(req, bodyBytes)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check if content is JSON
//	isJSON, _ := payload.IsJSON()
//
//	// Truncate large payloads
//	truncated, _ := payload.Truncate(1024)
//
//	// Use with slog
//	logger.Info("received request", "body", payload)
//
// # Features
//
//   - Automatic MIME type detection and handling
//   - Character encoding detection and UTF-8 conversion
//   - Base64 encoding for binary content
//   - Printability heuristics to distinguish text from binary
//   - Payload truncation for large bodies
//   - Integration with slog for structured logging
//   - JSON validation and pretty-printing
//
// # Content Detection
//
// The package uses multiple strategies to determine how to represent content:
//
// 1. MIME type checking (Content-Type header)
// 2. Character encoding detection (using chardet)
// 3. UTF-8 validation
// 4. Printability heuristics (95% printable characters threshold)
//
// Content that cannot be represented as text is automatically base64-encoded.
package printable

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/saintfish/chardet"
	"golang.org/x/net/html/charset"
)

// printabilityCheckLen defines how many bytes to check when determining if content is printable.
// This is a heuristic to avoid processing very large bodies unnecessarily. Only the first 1024 bytes
// are checked - if 95% of those are printable, the entire content is assumed to be printable.
const printabilityCheckLen = 1024

// Payload represents a payload that can be printed or displayed.
// It contains the content, its length, and whether it is base64 encoded.
// It also includes a truncated length for cases where the content is too large.
type Payload struct {
	Base64          bool   `json:"base64,omitempty"`
	Content         string `json:"content"`
	Length          int64  `json:"length"`
	TruncatedLength int64  `json:"truncatedLength,omitempty"`
}

// String returns the content as a string. If the payload is nil or empty, returns
// an empty string or "<nil>". This implements the fmt.Stringer interface.
func (p *Payload) String() string {
	if p == nil {
		return "<nil>"
	}

	if p.IsEmpty() {
		return ""
	}

	if p.IsTruncated() {
		return p.Content + "…"
	}

	return p.Content
}

// LogValue implements slog.LogValuer to provide rich structured logging of payloads.
// It returns a slog.GroupValue containing the raw content, parsed JSON (if applicable),
// base64 encoding flag, size information, and truncation details.
func (p *Payload) LogValue() slog.Value {
	if p == nil {
		return slog.StringValue("<nil>")
	}

	var attrs []slog.Attr

	attrs = append(attrs, slog.String("raw", p.String()))

	isJSON, err := p.IsJSON()
	if err == nil && isJSON {
		contentBytes, err := p.GetContentBytes()
		if err == nil && len(contentBytes) > 0 {
			var jsonValue any
			if err := json.Unmarshal(contentBytes, &jsonValue); err != nil {
				return slog.StringValue(p.String())
			}

			val := jsonToSlogValue(jsonValue)
			attrs = append(attrs, slog.Any("json", val))
		}
	}

	attrs = append(attrs, slog.Bool("base64", p.IsBase64()))
	attrs = append(attrs, slog.Int64("size", p.Length))

	if p.IsTruncated() {
		attrs = append(attrs, slog.Int64("sizeTruncated", p.GetTruncatedLength()))
	}

	return slog.GroupValue(attrs...)
}

// jsonToSlogValue recursively converts arbitrary JSON values into slog.Value types
// for structured logging. Maps become slog groups, arrays become indexed groups,
// and primitive types are converted to their appropriate slog value types.
func jsonToSlogValue(v any) slog.Value { //nolint:cyclop
	switch value := v.(type) {
	case map[string]any:
		attrs := make([]slog.Attr, 0, len(value))

		for k, val := range value {
			attrs = append(attrs, slog.Attr{
				Key:   k,
				Value: jsonToSlogValue(val),
			})
		}

		return slog.GroupValue(attrs...)
	case []any:
		attrs := make([]slog.Attr, len(value))

		for i, val := range value {
			attrs[i] = slog.Attr{
				Key:   strconv.FormatInt(int64(i), 10),
				Value: jsonToSlogValue(val),
			}
		}

		return slog.GroupValue(attrs...)
	case string:
		return slog.StringValue(value)
	case float32:
		return slog.Float64Value(float64(value)) // use Float64Value for consistency
	case float64:
		return slog.Float64Value(value)
	case int:
		return slog.Int64Value(int64(value)) // use Int64Value for consistency
	case int32:
		return slog.Int64Value(int64(value))
	case uint32:
		return slog.Uint64Value(uint64(value))
	case int64:
		return slog.Int64Value(value)
	case uint64:
		return slog.Uint64Value(value)
	case bool:
		return slog.BoolValue(value)
	case nil:
		return slog.AnyValue(nil)
	default:
		// fallback for unexpected types, or custom structs
		return slog.AnyValue(value)
	}
}

// IsEmpty returns true if the payload is nil or has no content.
func (p *Payload) IsEmpty() bool {
	return p == nil || (p.Content == "" && p.Length == 0)
}

// IsBase64 returns true if the content is base64-encoded (indicating binary data).
func (p *Payload) IsBase64() bool {
	return p != nil && p.Base64
}

// IsJSON checks if the payload content is valid JSON. Returns false if the
// payload is nil or if the content cannot be decoded as valid UTF-8.
func (p *Payload) IsJSON() (bool, error) {
	if p == nil {
		return false, nil
	}

	bts, err := p.GetContentBytes()
	if err != nil {
		return false, fmt.Errorf("error getting content bytes: %w", err)
	}

	return json.Valid(bts), nil
}

// GetContent returns the raw content string. Returns empty string if payload is nil.
func (p *Payload) GetContent() string {
	if p == nil {
		return ""
	}

	return p.Content
}

// GetContentBytes returns the content as a byte slice. If the content is base64-encoded,
// it is automatically decoded. Returns nil if the payload is nil.
func (p *Payload) GetContentBytes() ([]byte, error) {
	if p == nil {
		return nil, nil //nolint:nilnil
	}

	if p.IsBase64() {
		return base64.StdEncoding.DecodeString(p.Content)
	}

	return []byte(p.Content), nil
}

// GetLength returns the original content length in bytes. Returns 0 if payload is nil.
func (p *Payload) GetLength() int64 {
	if p == nil {
		return 0
	}

	return p.Length
}

// IsTruncated returns true if the content has been truncated to a smaller size.
func (p *Payload) IsTruncated() bool {
	if p == nil {
		return false
	}

	return p.GetTruncatedLength() < p.Length
}

// Clone creates a deep copy of the payload. Returns nil if the original is nil.
func (p *Payload) Clone() *Payload {
	if p == nil {
		return nil
	}

	return &Payload{
		Base64:          p.Base64,
		Content:         p.Content,
		Length:          p.Length,
		TruncatedLength: p.TruncatedLength,
	}
}

// Truncate returns a new payload with content truncated to the specified size in bytes.
// If the payload is already smaller than the specified size, it returns the original unchanged.
// For base64-encoded content, the underlying binary data is truncated before re-encoding.
// Returns nil if the payload is nil or size is negative.
func (p *Payload) Truncate(size int64) (*Payload, error) {
	if p == nil || size < 0 {
		return nil, nil //nolint:nilnil
	}

	if size >= p.Length || size >= p.GetTruncatedLength() {
		// No truncation needed, just return the original
		return p, nil
	}

	cloned := p.Clone()

	if p.IsBase64() {
		bts, err := p.GetContentBytes()
		if err != nil {
			return nil, fmt.Errorf("error getting content bytes: %w", err)
		}

		cloned.TruncatedLength = size
		truncated := bts[:size]
		cloned.Content = base64.StdEncoding.EncodeToString(truncated)
	} else {
		cloned.Content = cloned.Content[:size]

		// String truncation vs byte truncation may disagree in length (due
		// to multibyte characters), so we need to ensure the length is correct.
		cloned.TruncatedLength = int64(len([]byte(cloned.Content)))
	}

	return cloned, nil
}

// GetTruncatedLength returns the truncated content length in bytes.
// If the content has not been truncated, returns the full length.
func (p *Payload) GetTruncatedLength() int64 {
	if p == nil {
		return 0
	}

	if p.TruncatedLength > 0 {
		return p.TruncatedLength
	}

	// If not set, use the full length
	return p.Length
}

// Request creates a Payload from an HTTP request by converting its body to a printable format.
// The body parameter can be nil, in which case it will read from req.Body. If provided,
// the body bytes are used instead of reading from req.Body, which is useful when the body
// has already been read or needs to be preserved for other uses.
func Request(req *http.Request, body []byte) (*Payload, error) {
	return getBodyAsPrintable(&requestContentReader{
		Request:   req,
		BodyBytes: body,
	})
}

// Response creates a Payload from an HTTP response by converting its body to a printable format.
// The body parameter can be nil, in which case it will read from resp.Body. If provided,
// the body bytes are used instead of reading from resp.Body, which is useful when the body
// has already been read or needs to be preserved for other uses.
func Response(resp *http.Response, body []byte) (*Payload, error) {
	return getBodyAsPrintable(&responseContentReader{
		Response:  resp,
		BodyBytes: body,
	})
}

// requestContentReader is an internal adapter for reading HTTP request bodies.
// It implements the bodyContentReader interface and provides access to request
// headers and body content, with support for pre-read body bytes.
type requestContentReader struct {
	Request   *http.Request
	BodyBytes []byte
}

// GetBody returns an io.ReadCloser for the request body. If BodyBytes is set,
// it returns a reader for those bytes. Otherwise, it returns the original req.Body.
func (r *requestContentReader) GetBody() io.ReadCloser {
	if r.Request == nil {
		return nil
	}

	if r.BodyBytes != nil {
		return io.NopCloser(bytes.NewReader(r.BodyBytes))
	}

	return r.Request.Body
}

// GetHeaders returns the HTTP request headers.
func (r *requestContentReader) GetHeaders() http.Header {
	if r.Request == nil {
		return nil
	}

	return r.Request.Header
}

// SetBody updates the request body with a new io.ReadCloser. This clears any
// cached BodyBytes since the body is being replaced.
func (r *requestContentReader) SetBody(body io.ReadCloser) {
	if r.Request == nil {
		return
	}

	r.BodyBytes = nil // Clear cached bytes if we set a new body
	r.Request.Body = body
}

// responseContentReader is an internal adapter for reading HTTP response bodies.
// It implements the bodyContentReader interface and provides access to response
// headers and body content, with support for pre-read body bytes.
type responseContentReader struct {
	Response  *http.Response
	BodyBytes []byte
}

// GetBody returns an io.ReadCloser for the response body. If BodyBytes is set,
// it returns a reader for those bytes. Otherwise, it returns the original resp.Body.
func (r *responseContentReader) GetBody() io.ReadCloser {
	if r.Response == nil {
		return nil
	}

	if r.BodyBytes != nil {
		return io.NopCloser(bytes.NewReader(r.BodyBytes))
	}

	return r.Response.Body
}

// GetHeaders returns the HTTP response headers.
func (r *responseContentReader) GetHeaders() http.Header {
	if r.Response == nil {
		return nil
	}

	return r.Response.Header
}

// SetBody updates the response body with a new io.ReadCloser. This clears any
// cached BodyBytes since the body is being replaced.
func (r *responseContentReader) SetBody(body io.ReadCloser) {
	if r.Response == nil {
		return
	}

	r.Response.Body = body
	r.BodyBytes = nil
}

// bodyContentReader is an internal interface that abstracts access to HTTP
// request or response bodies and headers. It allows the same body processing
// logic to work with both requests and responses.
type bodyContentReader interface {
	GetBody() io.ReadCloser
	GetHeaders() http.Header
	SetBody(body io.ReadCloser)
}

// isPrintableMimeType checks if a MIME type represents text-based or printable content.
// Returns true for text/*, application/json, application/xml, and other known text formats.
func isPrintableMimeType(mimeType string) bool {
	// Check if the MIME type is text-based or a known printable format
	return strings.HasPrefix(mimeType, "text/") ||
		strings.HasSuffix(mimeType, "+json") ||
		strings.HasSuffix(mimeType, "+xml") ||
		mimeType == "application/json" ||
		mimeType == "application/xml" ||
		mimeType == "application/javascript" ||
		mimeType == "application/x-www-form-urlencoded"
}

// peekBody reads the entire body content without consuming it. It uses io.TeeReader
// to read the body while simultaneously writing to a buffer, then restores the body
// for further use. This allows body inspection without preventing subsequent reads.
func peekBody(bcr bodyContentReader) ([]byte, error) {
	if bcr == nil || bcr.GetBody() == nil {
		return nil, nil
	}

	body := bcr.GetBody()

	// Read the body without closing it
	var buf bytes.Buffer

	tee := io.TeeReader(body, &buf)

	data, err := io.ReadAll(tee)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Restore the body for further use
	bcr.SetBody(io.NopCloser(&buf))

	return data, nil
}

// getBodyAsPrintable does what it can to convert the body of a request or response
// into a printable format. It checks the MIME type, charset, and content to determine
// if the content is printable or needs to be base64 encoded. It also handles UTF-8 decoding
// and checks for printability using a heuristic based on the proportion of printable characters.
func getBodyAsPrintable(bcr bodyContentReader) (*Payload, error) {
	if bcr == nil {
		return nil, nil //nolint:nilnil
	}

	rawData, mimeType, charsetStr, err := getDataAndMimeType(bcr)
	if err != nil {
		return nil, err
	}

	if len(rawData) == 0 {
		return nil, nil //nolint:nilnil
	}

	if mimeType != "" && !isPrintableMimeType(mimeType) {
		return &Payload{
			Base64:  true,
			Content: base64.StdEncoding.EncodeToString(rawData),
			Length:  int64(len(rawData)),
		}, nil
	}

	decodedData, isUtf8, err := getDataAsUtf8(rawData, charsetStr)
	if err != nil {
		return nil, fmt.Errorf("error decoding body as UTF-8: %w", err)
	}

	// If we can't get utf-8, we have no choice but to treat it as binary
	if !isUtf8 {
		return &Payload{
			Base64:  true,
			Content: base64.StdEncoding.EncodeToString(rawData),
			Length:  int64(len(rawData)),
		}, nil
	}

	checkLen := len(decodedData)

	if checkLen > printabilityCheckLen {
		checkLen = printabilityCheckLen
	}

	sample := decodedData[:checkLen]

	printable, total := 0, 0

	for len(sample) > 0 {
		r, size := utf8.DecodeRune(sample)

		sample = sample[size:]
		total++

		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			printable++
		}
	}

	if total == 0 {
		return &Payload{
			Content: "",
			Length:  0,
		}, nil
	}

	// Heuristic: 95%+ means printable
	isPrintable := float64(printable)/float64(total) > 0.95 //nolint:mnd

	if isPrintable {
		return &Payload{
			Content: string(decodedData),
			Length:  int64(len(decodedData)),
		}, nil
	}

	return &Payload{
		Base64:  true,
		Content: base64.StdEncoding.EncodeToString(rawData),
		Length:  int64(len(rawData)),
	}, nil
}

// getDataAndMimeType extracts the body data, MIME type, and charset from a request or response.
// It parses the Content-Type header to extract MIME type and charset parameters, and reads
// the body content using peekBody to avoid consuming the body stream.
func getDataAndMimeType(bcr bodyContentReader) (data []byte, mimeType string, charset string, err error) {
	if bcr == nil {
		return nil, "", "", nil
	}

	// Check MIME type
	contentType := bcr.GetHeaders().Get("Content-Type")

	mimeType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// If parsing fails, fallback to sniffing the content
		mimeType = ""
	}

	charsetStr := ""
	if cs, ok := params["charset"]; ok {
		charsetStr = strings.ToLower(cs)
	}

	rawData, err := peekBody(bcr)
	if err != nil {
		return nil, "", "", fmt.Errorf("error peeking response body: %w", err)
	}

	if rawData == nil {
		return nil, "", "", nil
	}

	return rawData, mimeType, charsetStr, nil
}

// getDataAsUtf8 attempts to decode the given data as UTF-8 using the provided charset hint.
// It uses golang.org/x/net/html/charset to convert from the source charset to UTF-8.
// Returns the decoded data, a boolean indicating if the result is valid UTF-8, and any error.
//
// Note: Even valid UTF-8 may contain control characters or invisible characters that are
// not printable. The caller should perform additional printability checks if needed.
func getDataAsUtf8(data []byte, charsetStr string) ([]byte, bool, error) {
	// Get a UTF-8 reader, either using the provided charset or by detecting it
	decodedReader, _ := getUtf8Reader(data, charsetStr)

	// Normalize the input to UTF-8, hopefully
	decodedData, err := io.ReadAll(decodedReader)
	if err != nil {
		return nil, false, err
	}

	// Check UTF-8 validity (paranoia)
	if !utf8.Valid(decodedData) {
		return data, false, nil
	}

	// Return the decoded data
	return decodedData, true, nil
}

// getUtf8Reader creates an io.Reader that decodes the data to UTF-8.
// It first tries to use the provided charset hint. If that fails or no charset
// is provided, it uses github.com/saintfish/chardet to detect the charset automatically.
// Returns the reader and the charset name that was used (provided or detected).
func getUtf8Reader(data []byte, charsetStr string) (io.Reader, string) {
	// First try with the provided charset
	decodedReader, err := charset.NewReaderLabel(charsetStr, bytes.NewReader(data))
	if err == nil {
		// Success
		return decodedReader, charsetStr
	}

	// If that fails, try to detect the charset
	detector := chardet.NewTextDetector()

	best, err := detector.DetectBest(data)
	if err != nil {
		// Last resort, assume UTF-8, even if it might be wrong
		return bytes.NewReader(data), "utf-8"
	}

	// We have a detected charset, try to use it
	decodedReader, err = charset.NewReaderLabel(best.Charset, bytes.NewReader(data))
	if err != nil {
		// Last resort, assume UTF-8, even if it might be wrong
		decodedReader = bytes.NewReader(data)
	}

	// Return the detected charset and reader
	return decodedReader, best.Charset
}
