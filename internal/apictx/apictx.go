package apictx

import (
	"context"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/s3api/actions"
)

type ctxKey struct{}

// APICtx contains API request details extracted from the HTTP request.
type APICtx struct {
	Method        string
	URI           string
	Path          string
	QueryParams   url.Values
	Host          string
	Scheme        string
	RemoteAddr    string
	UserAgent     string
	RequestID     string
	ContentType   string
	ContentLength int64
	Headers       http.Header

	Action   actions.Action
	Username *string
	Bucket   core.Bucket
}

// Inject adds ApiCtx to the context and returns a new context.
// The ApiCtx is extracted from the Echo context.
func Inject(c *echo.Context) context.Context {
	req := c.Request()

	apiCtx := APICtx{
		Method:        req.Method,
		URI:           req.RequestURI,
		Path:          req.URL.Path,
		QueryParams:   req.URL.Query(),
		Host:          req.Host,
		Scheme:        getScheme(req),
		RemoteAddr:    req.RemoteAddr,
		UserAgent:     req.UserAgent(),
		RequestID:     getRequestID(req),
		ContentType:   req.Header.Get("Content-Type"),
		ContentLength: req.ContentLength,
		Headers:       req.Header,
	}

	return context.WithValue(req.Context(), ctxKey{}, &apiCtx)
}

// FromContext retrieves ApiCtx from the context.
// Returns nil if ApiCtx is not present in the context.
func FromContext(ctx context.Context) *APICtx {
	apiCtx, ok := ctx.Value(ctxKey{}).(*APICtx)
	if !ok {
		return nil
	}

	return apiCtx
}

// MustFromContext retrieves ApiCtx from the context.
// Panics if ApiCtx is not present in the context.
func MustFromContext(ctx context.Context) *APICtx {
	apiCtx := FromContext(ctx)
	if apiCtx == nil {
		panic("ApiCtx not found in context")
	}

	return apiCtx
}

// getScheme determines the request scheme, checking X-Forwarded-Proto header first.
func getScheme(req *http.Request) string {
	if scheme := req.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}

	if req.TLS != nil {
		return "https"
	}

	return "http"
}

// getRequestID extracts request ID from common header names.
func getRequestID(req *http.Request) string {
	for _, header := range []string{"Amz-Sdk-Invocation-Id", "X-Request-ID", "X-Request-Id", "Request-ID"} {
		if id := req.Header.Get(header); id != "" {
			return id
		}
	}

	return ""
}
