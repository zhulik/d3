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

func Validate(ctx context.Context, r *http.Request, accessKeyResolver AccessKeyResolver) (string, error) {
	keyID, err := validate(ctx, r.Method, r.URL, r.Header, r.Host, accessKeyResolver)
	if errors.Is(err, ErrSignatureDoesNotMatch) {
		// we want to retry the validation with a trailing slash

		// if the path already has a trailing slash, we return the error as is
		if strings.HasSuffix(r.URL.Path, "/") {
			return "", err
		}

		// if the path doesn't have a trailing slash, we add it and retry the validation
		newURL := *r.URL
		newURL.Path += "/"

		return validate(ctx, r.Method, &newURL, r.Header, r.Host, accessKeyResolver)
	}

	return keyID, nil
}

func validate(ctx context.Context, method string, u *url.URL, header http.Header, host string, accessKeyResolver AccessKeyResolver) (string, error) { //nolint:lll
	hp, err := extractAuthHeaderParameters(u, header)
	if err != nil {
		return "", err
	}

	canURI := buildCanonicalURI(u)
	canQuery := buildCanonicalQueryString(u)
	canHeaders := buildCanonicalHeaders(hp.signedHeaders, header, host)

	canReq := strings.Join([]string{
		method,
		canURI,
		canQuery,
		canHeaders,
		strings.ToLower(hp.signedHeaders),
		hp.hashedPayload,
	}, "\n")

	hash := sha256.Sum256([]byte(canReq))
	canReqHashHex := hex.EncodeToString(hash[:])

	scope := buildSigningScope(hp.scopeRegion, hp.scopeService, hp.scopeDate)
	sts := strings.Join([]string{
		AlgoHMAC256,
		hp.requestTime.Format(TimeFormat),
		scope,
		canReqHashHex,
	}, "\n")

	secretKey, err := accessKeyResolver(ctx, hp.accessKey)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidAccessKeyID, err)
	}

	kSigning := deriveSigningKey(hp.scopeRegion, hp.scopeService, secretKey, hp.scopeDate)

	sigBytes := hmacSHA256(kSigning, sts)
	calcSig := hex.EncodeToString(sigBytes)

	if !hmac.Equal([]byte(hp.signature), []byte(calcSig)) {
		return "", ErrSignatureDoesNotMatch
	}

	return hp.accessKey, nil
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
		algo:          AlgoHMAC256,
		hashedPayload: header.Get("X-Amz-Content-Sha256"),
		requestTime:   requestTime,
	}

	if headerParams.requestTime.IsZero() {
		return nil, ErrMissingDateHeader
	}

	if headerParams.hashedPayload == "" {
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
				headerParams.accessKey = credParts[0]

				scopeDate, err := time.Parse(ShortTimeFormat, credParts[1])
				if err != nil {
					return nil, err
				}

				headerParams.scopeDate = scopeDate
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

func extractAuthHeadersParamsFromSignedURL(u *url.URL) (*AuthHeaderParameters, error) {
	qs := u.Query()

	requestTime, err := time.Parse(TimeFormat, qs.Get("X-Amz-Date"))
	if err != nil {
		return nil, err
	}

	headerParams := &AuthHeaderParameters{
		algo:          AlgoHMAC256,
		requestTime:   requestTime,
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

		scopeDate, err := time.Parse(ShortTimeFormat, credParts[1])
		if err != nil {
			return nil, err
		}

		headerParams.scopeDate = scopeDate
		headerParams.scopeRegion = credParts[2]
		headerParams.scopeService = credParts[3]
	}

	if expStr := qs.Get("X-Amz-Expires"); expStr != "" {
		maxT := headerParams.requestTime.Add(time.Duration(parseIntDefault(expStr, 0)) * time.Second).Add(5 * time.Minute)
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
