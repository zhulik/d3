package sigv4

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"strings"
	"time"
)

const (
	// precalculated hash for the zero chunk length.
	emptyChunkSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// ChunkSigner implements signing of aws-chunked payloads.
type ChunkSigner struct {
	region  string
	service string

	accessKeyID     string
	secretAccessKey string

	prevSig  []byte
	seedDate time.Time
}

// NewChunkSigner creates a SigV4 signer used to sign Event Stream encoded messages.
func NewChunkSigner(region, service string, seedSignature []byte, seedDate time.Time, accessKeyID, secretAccessKey string) *ChunkSigner { //nolint:lll
	return &ChunkSigner{
		region:          region,
		service:         service,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		seedDate:        seedDate,
		prevSig:         seedSignature,
	}
}

// GetSignature takes an event stream encoded headers and payload and returns a signature.
func (s *ChunkSigner) GetSignature(payload []byte) []byte {
	sum := sha256.Sum256(payload)

	return s.getSignature(sum[:])
}

// GetSignatureByHash takes an event stream encoded headers and payload and returns a signature.
func (s *ChunkSigner) GetSignatureByHash(payloadHash hash.Hash) []byte {
	return s.getSignature(payloadHash.Sum(nil))
}

func (s *ChunkSigner) getSignature(payloadHash []byte) []byte {
	sigKey := deriveSigningKey(s.region, s.service, s.secretAccessKey, s.seedDate)

	keyPath := buildSigningScope(s.region, s.service, s.seedDate)

	stringToSign := buildStringToSign(payloadHash, s.prevSig, keyPath, s.seedDate)

	signature := hmacSHA256(sigKey, stringToSign)
	s.prevSig = signature

	return signature
}

func buildStringToSign(payloadHash, prevSig []byte, scope string, date time.Time) string {
	return strings.Join([]string{
		"AWS4-HMAC-SHA256-PAYLOAD",
		date.UTC().Format(TimeFormat),
		scope,
		hex.EncodeToString(prevSig),
		emptyChunkSHA256,
		hex.EncodeToString(payloadHash),
	}, "\n")
}

func buildSigningScope(region, service string, dt time.Time) string {
	return strings.Join([]string{
		dt.UTC().Format(ShortTimeFormat),
		region,
		service,
		ScopeAWS4Request,
	}, "/")
}

func deriveSigningKey(region, service, secretKey string, dt time.Time) []byte {
	hmacDate := hmacSHA256([]byte("AWS4"+secretKey), dt.UTC().Format(ShortTimeFormat))
	hmacRegion := hmacSHA256(hmacDate, region)
	hmacService := hmacSHA256(hmacRegion, service)
	signingKey := hmacSHA256(hmacService, ScopeAWS4Request)

	return signingKey
}
