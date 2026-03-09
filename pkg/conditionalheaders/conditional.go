package conditionalheaders

import (
	"net/http"
	"strings"
	"time"
)

// Conditionals holds parsed conditional request headers (RFC 7232 / S3).
// Used by GetObject and HeadObject. Empty string means the header was not present.
// IfModifiedSince and IfUnmodifiedSince are nil when absent or unparseable.
type Conditionals struct {
	IfMatch           string     // raw header value, empty if absent
	IfNoneMatch       string     // raw header value; "*" means match any ETag
	IfModifiedSince   *time.Time // nil if absent or invalid
	IfUnmodifiedSince *time.Time // nil if absent or invalid
}

// Parse parses If-Match, If-None-Match, If-Modified-Since, and If-Unmodified-Since from headers.
func Parse(headers http.Header) Conditionals {
	var c Conditionals
	if v := headers.Get("If-Match"); v != "" {
		c.IfMatch = strings.TrimSpace(v)
	}

	if v := headers.Get("If-None-Match"); v != "" {
		c.IfNoneMatch = strings.TrimSpace(v)
	}

	if v := headers.Get("If-Modified-Since"); v != "" {
		if t, ok := ParseHTTPDate(v); ok {
			c.IfModifiedSince = &t
		}
	}

	if v := headers.Get("If-Unmodified-Since"); v != "" {
		if t, ok := ParseHTTPDate(v); ok {
			c.IfUnmodifiedSince = &t
		}
	}

	return c
}

// Check evaluates the conditionals against the object's ETag and LastModified.
// Returns the HTTP status code: 200 (proceed), 304 (Not Modified), or 412 (Precondition Failed).
func (c Conditionals) Check(objectETag string, lastModified time.Time) int {
	if c.IfMatch != "" {
		if !etagMatchList(c.IfMatch, objectETag) {
			return http.StatusPreconditionFailed
		}
	}

	if c.IfNoneMatch != "" {
		if c.IfNoneMatch == "*" || etagMatchList(c.IfNoneMatch, objectETag) {
			return http.StatusNotModified
		}
	}

	if c.IfModifiedSince != nil {
		if !lastModified.After(*c.IfModifiedSince) {
			return http.StatusNotModified
		}
	}

	if c.IfUnmodifiedSince != nil {
		if lastModified.After(*c.IfUnmodifiedSince) {
			return http.StatusPreconditionFailed
		}
	}

	return http.StatusOK
}

func etagMatchList(headerValue, objectETag string) bool {
	for s := range strings.SplitSeq(headerValue, ",") {
		if ETagMatches(strings.TrimSpace(s), objectETag) {
			return true
		}
	}

	return false
}

// ETagMatches reports whether the client ETag (e.g. from a header) matches the object ETag.
// Both are normalized: optional W/ prefix and surrounding quotes are stripped (RFC 7232).
func ETagMatches(clientETag, objectETag string) bool {
	return NormalizeETag(clientETag) == NormalizeETag(objectETag)
}

// NormalizeETag strips optional W/ prefix and surrounding quotes for strong comparison.
func NormalizeETag(etag string) string {
	s := strings.TrimSpace(etag)
	if strings.HasPrefix(s, "W/") {
		s = strings.TrimSpace(s[2:])
	}

	return strings.Trim(s, "\"")
}

// ParseHTTPDate parses an HTTP-date (RFC 7231). Returns (zero time, false) on parse error.
func ParseHTTPDate(s string) (time.Time, bool) {
	t, err := http.ParseTime(strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}
