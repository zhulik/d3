package sigv4

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/samber/lo"
)

type AuthHeaderParameters struct {
	Algo          string
	AccessKey     string `json:"-"`
	ScopeDate     time.Time
	ScopeRegion   string
	ScopeService  string
	SignedHeaders string
	Signature     []byte
	RequestTime   time.Time
	HashedPayload string
}

func (hp *AuthHeaderParameters) Validate() error {
	if hp.Algo != "AWS4-HMAC-SHA256" || hp.AccessKey == "" || hp.SignedHeaders == "" ||
		len(hp.Signature) == 0 || hp.RequestTime.IsZero() {
		return fmt.Errorf("%w: signature or request time missing", ErrSignatureDoesNotMatch)
	}

	// Minimal validation of scope
	if hp.ScopeDate.IsZero() || hp.ScopeRegion == "" || hp.ScopeService != "s3" {
		return ErrCredMalformed
	}

	// Validate time skew (+/- 5 minutes)
	if d := time.Since(hp.RequestTime.UTC()); d > 5*time.Minute || d < -5*time.Minute {
		return ErrRequestNotReadyYet
	}

	return nil
}

func (hp *AuthHeaderParameters) RawSignature() []byte {
	return lo.Must(hex.DecodeString(string(hp.Signature)))
}
