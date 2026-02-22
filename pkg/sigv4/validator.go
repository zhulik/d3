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
	TimeFormat       = "20060102T150405Z"
	ShortTimeFormat  = "20060102"
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

type AccessKeyResolver func(ctx context.Context, accessKey string) (string, error)

func Validate(ctx context.Context, r *http.Request, accessKeyResolver AccessKeyResolver) (*AuthHeaderParameters, error) { //nolint:lll
	keyID, err := validate(ctx, r.Method, r.URL, r.Header, r.Host, accessKeyResolver)
	if errors.Is(err, ErrSignatureDoesNotMatch) {
		// we want to retry the validation with a trailing slash

		// if the path already has a trailing slash, we return the error as is
		if strings.HasSuffix(r.URL.Path, "/") {
			return nil, err
		}

		// if the path doesn't have a trailing slash, we add it and retry the validation
		newURL := *r.URL
		newURL.Path += "/"

		return validate(ctx, r.Method, &newURL, r.Header, r.Host, accessKeyResolver)
	}

	return keyID, nil
}

func validate(ctx context.Context, method string, u *url.URL, header http.Header, host string, accessKeyResolver AccessKeyResolver) (*AuthHeaderParameters, error) { //nolint:lll
	hp, err := extractAuthHeaderParameters(u, header)
	if err != nil {
		return nil, err
	}

	canURI := buildCanonicalURI(u)
	canQuery := buildCanonicalQueryString(u)
	canHeaders := buildCanonicalHeaders(hp.SignedHeaders, header, host)

	canReq := strings.Join([]string{
		method,
		canURI,
		canQuery,
		canHeaders,
		strings.ToLower(hp.SignedHeaders),
		hp.HashedPayload,
	}, "\n")

	hash := sha256.Sum256([]byte(canReq))
	canReqHashHex := hex.EncodeToString(hash[:])

	scope := buildSigningScope(hp.ScopeRegion, hp.ScopeService, hp.ScopeDate)
	sts := strings.Join([]string{
		AlgoHMAC256,
		hp.RequestTime.Format(TimeFormat),
		scope,
		canReqHashHex,
	}, "\n")

	secretKey, err := accessKeyResolver(ctx, hp.AccessKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidAccessKeyID, err)
	}

	kSigning := deriveSigningKey(hp.ScopeRegion, hp.ScopeService, secretKey, hp.ScopeDate)

	sigBytes := hmacSHA256(kSigning, sts)
	calcSig := hex.EncodeToString(sigBytes)

	if !hmac.Equal(hp.Signature, []byte(calcSig)) {
		return nil, ErrSignatureDoesNotMatch
	}

	return hp, nil
}

func extractAuthHeaderParameters(u *url.URL, header http.Header) (*AuthHeaderParameters, error) {
	var headerParams *AuthHeaderParameters

	qs := u.Query()

	var err error

	if auth := header.Get("Authorization"); strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") {
		headerParams, err = extractAuthHeadersParamsFromAuthHeader(header)
		if err != nil {
			return nil, err
		}
	} else if qs.Get("X-Amz-Algorithm") == AlgoHMAC256 {
		headerParams, err = extractAuthHeadersParamsFromSignedURL(u)
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

func extractAuthHeadersParamsFromAuthHeader(header http.Header) (*AuthHeaderParameters, error) {
	auth := header.Get("Authorization")

	requestTime, err := time.Parse(TimeFormat, header.Get("X-Amz-Date"))
	if err != nil {
		return nil, err
	}

	headerParams := &AuthHeaderParameters{
		Algo:          AlgoHMAC256,
		HashedPayload: header.Get("X-Amz-Content-Sha256"),
		RequestTime:   requestTime,
	}

	if headerParams.RequestTime.IsZero() {
		return nil, ErrMissingDateHeader
	}

	if headerParams.HashedPayload == "" {
		return nil, ErrInvalidDigest
	}

	parts := strings.SplitSeq(auth[len("AWS4-HMAC-SHA256 "):], ",")
	for part := range parts {
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
				headerParams.AccessKey = credParts[0]

				scopeDate, err := time.Parse(ShortTimeFormat, credParts[1])
				if err != nil {
					return nil, err
				}

				headerParams.ScopeDate = scopeDate
				headerParams.ScopeRegion = credParts[2]
				headerParams.ScopeService = credParts[3]
			}
		case "SignedHeaders":
			headerParams.SignedHeaders = val
		case "Signature":
			headerParams.Signature = []byte(strings.ToLower(val))
		}
	}

	return headerParams, nil
}

func extractAuthHeadersParamsFromSignedURL(u *url.URL) (*AuthHeaderParameters, error) {
	qs := u.Query()

	requestTime, err := time.Parse(TimeFormat, qs.Get("X-Amz-Date"))
	if err != nil {
		return nil, err
	}

	headerParams := &AuthHeaderParameters{
		Algo:          AlgoHMAC256,
		RequestTime:   requestTime,
		SignedHeaders: qs.Get("X-Amz-SignedHeaders"),
		Signature:     []byte(strings.ToLower(qs.Get("X-Amz-Signature"))),
		HashedPayload: qs.Get("X-Amz-Content-Sha256"),
	}

	if headerParams.HashedPayload == "" {
		headerParams.HashedPayload = "UNSIGNED-PAYLOAD"
	}

	cred := qs.Get("X-Amz-Credential")

	credParts := strings.Split(cred, "/")
	if len(credParts) >= 5 {
		headerParams.AccessKey = credParts[0]

		scopeDate, err := time.Parse(ShortTimeFormat, credParts[1])
		if err != nil {
			return nil, err
		}

		headerParams.ScopeDate = scopeDate
		headerParams.ScopeRegion = credParts[2]
		headerParams.ScopeService = credParts[3]
	}

	if expStr := qs.Get("X-Amz-Expires"); expStr != "" {
		maxT := headerParams.RequestTime.Add(time.Duration(parseIntDefault(expStr, 0)) * time.Second).Add(5 * time.Minute)
		if time.Now().UTC().After(maxT) {
			return nil, ErrExpiredPresignRequest
		}
	}

	return headerParams, nil
}

func buildCanonicalURI(url *url.URL) string {
	canURI := url.EscapedPath()
	if canURI == "" {
		canURI = "/"
	}

	return canURI
}

func buildCanonicalQueryString(u *url.URL) string {
	var canQuery string

	qp := url.Values{}

	for k, vs := range u.Query() {
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

func buildCanonicalHeaders(signedHeaders string, header http.Header, host string) string {
	sh := strings.Split(signedHeaders, ";")
	sort.Strings(sh)

	var canonicalHeaders strings.Builder

	for _, h := range sh {
		name := strings.ToLower(strings.TrimSpace(h))

		val := strings.TrimSpace(header.Get(name))
		if name == "host" && val == "" {
			val = host
		}

		val = collapseSpaces(val)

		canonicalHeaders.WriteString(fmt.Sprintf("%s:%s\n", name, val))
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

	for i := range len(s) {
		c := s[i]

		isUnreserved := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~'
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

	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return def
		}

		n = n*10 + int(s[i]-'0')
	}

	return n
}
