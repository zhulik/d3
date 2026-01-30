package sigv4

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	AlgoHMAC256      = "AWS4-HMAC-SHA256"
	ScopeAWS4Request = "aws4_request"
)

var (
	ErrMissingDateHeader      = errors.New("missing date header")
	ErrInvalidDigest          = errors.New("invalid digest")
	ErrExpiredPresignRequest  = errors.New("expired presign request")
	ErrMalformedPresignedDate = errors.New("malformed presigned date")
	ErrAccessDenied           = errors.New("access denied")
	ErrSignatureDoesNotMatch  = errors.New("signature does not match")
	ErrCredMalformed          = errors.New("credential malformed")
	ErrRequestNotReadyYet     = errors.New("request not ready yet")
	ErrInvalidAccessKeyID     = errors.New("invalid access key ID")
)

type CredentialStore interface {
	Get(ctx context.Context, accessKey string) (string, error)
}

func Validate(ctx context.Context, r *http.Request, credentialStore CredentialStore) error {
	hp, err := extractAuthHeaderParameters(r)
	if err != nil {
		return err
	}

	canURI := buildCanonicalURI(r)
	canQuery := buildCanonicalQueryString(r)
	canHeaders := buildCanonicalHeaders(hp.signedHeaders, r)

	canReq := strings.Join([]string{
		r.Method,
		canURI,
		canQuery,
		canHeaders,
		strings.ToLower(hp.signedHeaders),
		hp.hashedPayload,
	}, "\n")

	hash := sha256.Sum256([]byte(canReq))
	canReqHashHex := hex.EncodeToString(hash[:])

	scope := strings.Join([]string{hp.scopeDate, hp.scopeRegion, hp.scopeService, ScopeAWS4Request}, "/")
	sts := strings.Join([]string{
		AlgoHMAC256,
		hp.requestTime,
		scope,
		canReqHashHex,
	}, "\n")

	secretKey, err := credentialStore.Get(ctx, hp.accessKey)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidAccessKeyID, err)
	}

	kDate := hmacSHA256([]byte("AWS4"+secretKey), hp.scopeDate)
	kRegion := hmacSHA256(kDate, hp.scopeRegion)
	kService := hmacSHA256(kRegion, hp.scopeService)
	kSigning := hmacSHA256(kService, ScopeAWS4Request)
	sigBytes := hmacSHA256(kSigning, sts)
	calcSig := hex.EncodeToString(sigBytes)

	if !hmac.Equal([]byte(hp.signature), []byte(calcSig)) {
		return ErrSignatureDoesNotMatch
	}

	return nil
}

func extractAuthHeaderParameters(r *http.Request) (*AuthHeaderParameters, error) {
	var headerParams *AuthHeaderParameters
	qs := r.URL.Query()
	var err error

	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") {
		headerParams, err = extractAuthHeadersParamsFromAuthHeader(r)
		if err != nil {
			return nil, err
		}
	} else if qs.Get("X-Amz-Algorithm") == AlgoHMAC256 {
		headerParams, err = extractAuthHeadersParamsFromSignedURL(r)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrAccessDenied
	}

	if err := headerParams.Validate(); err != nil {
		return nil, err
	}

	return headerParams, nil
}

func extractAuthHeadersParamsFromAuthHeader(r *http.Request) (*AuthHeaderParameters, error) {
	auth := r.Header.Get("Authorization")

	headerParams := &AuthHeaderParameters{
		algo:          AlgoHMAC256,
		hashedPayload: r.Header.Get("x-amz-content-sha256"),
		requestTime:   r.Header.Get("x-amz-date"),
	}

	if headerParams.requestTime == "" {
		return nil, ErrMissingDateHeader
	}
	if headerParams.hashedPayload == "" {
		return nil, ErrInvalidDigest
	}

	parts := strings.Split(auth[len("AWS4-HMAC-SHA256 "):], ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := kv[0]
		val := strings.Trim(kv[1], " ")
		switch key {
		case "Credential":
			credParts := strings.Split(val, "/")
			if len(credParts) >= 5 {
				headerParams.accessKey = credParts[0]
				headerParams.scopeDate = credParts[1]
				headerParams.scopeRegion = credParts[2]
				headerParams.scopeService = credParts[3]
			}
		case "SignedHeaders":
			headerParams.signedHeaders = val
		case "Signature":
			headerParams.signature = strings.ToLower(val)
		}
	}

	return headerParams, nil
}

func extractAuthHeadersParamsFromSignedURL(r *http.Request) (*AuthHeaderParameters, error) {
	qs := r.URL.Query()
	headerParams := &AuthHeaderParameters{
		algo:          AlgoHMAC256,
		requestTime:   qs.Get("X-Amz-Date"),
		signedHeaders: qs.Get("X-Amz-SignedHeaders"),
		signature:     strings.ToLower(qs.Get("X-Amz-Signature")),
		hashedPayload: qs.Get("X-Amz-Content-Sha256"),
	}

	if headerParams.hashedPayload == "" {
		headerParams.hashedPayload = "UNSIGNED-PAYLOAD"
	}

	cred := qs.Get("X-Amz-Credential")
	credParts := strings.Split(cred, "/")
	if len(credParts) >= 5 {
		headerParams.accessKey = credParts[0]
		headerParams.scopeDate = credParts[1]
		headerParams.scopeRegion = credParts[2]
		headerParams.scopeService = credParts[3]
	}

	if expStr := qs.Get("X-Amz-Expires"); expStr != "" {
		if tReq, err := time.Parse("20060102T150405Z", headerParams.requestTime); err == nil {
			maxT := tReq.Add(time.Duration(parseIntDefault(expStr, 0)) * time.Second).Add(5 * time.Minute)
			if time.Now().UTC().After(maxT) {
				return nil, ErrExpiredPresignRequest
			}
		} else {
			return nil, ErrMalformedPresignedDate
		}
	}
	return headerParams, nil
}

func buildCanonicalURI(r *http.Request) string {
	canURI := r.URL.EscapedPath()
	if canURI == "" {
		canURI = "/"
	}
	return canURI
}

func buildCanonicalQueryString(r *http.Request) string {
	var canQuery string
	qp := url.Values{}
	for k, vs := range r.URL.Query() {
		if k == "X-Amz-Signature" {
			continue
		}
		for _, v := range vs {
			qp.Add(k, v)
		}
	}
	var keys []string
	for k := range qp {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vals := qp[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, awsURLEncode(k, true)+"="+awsURLEncode(v, true))
		}
	}
	canQuery = strings.Join(parts, "&")

	return canQuery
}

func buildCanonicalHeaders(signedHeaders string, r *http.Request) string {
	sh := strings.Split(signedHeaders, ";")
	sort.Strings(sh)
	var canonicalHeaders strings.Builder
	for _, h := range sh {
		name := strings.ToLower(strings.TrimSpace(h))
		val := strings.TrimSpace(r.Header.Get(name))
		if name == "host" && val == "" {
			val = r.Host
		}
		val = collapseSpaces(val)
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(val)
		canonicalHeaders.WriteString("\n")
	}

	return canonicalHeaders.String()
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func collapseSpaces(s string) string {
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		} else {
			b.WriteRune(r)
			lastSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

func awsURLEncode(s string, encodeSlash bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		isUnreserved := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~'
		if isUnreserved || (!encodeSlash && c == '/') {
			b.WriteByte(c)
		} else {
			b.WriteString("%")
			b.WriteString(strings.ToUpper(hex.EncodeToString([]byte{c})))
		}
	}
	return b.String()
}

func parseIntDefault(s string, def int) int {
	var n int
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return def
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}
