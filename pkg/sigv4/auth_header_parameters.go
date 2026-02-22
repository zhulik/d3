package sigv4

import (
	"fmt"
	"time"
)

type AuthHeaderParameters struct {
	algo          string
	accessKey     string
	scopeDate     time.Time
	scopeRegion   string
	scopeService  string
	signedHeaders string
	signature     string
	requestTime   time.Time
	hashedPayload string
}

func (hp *AuthHeaderParameters) Validate() error {
	if hp.algo != "AWS4-HMAC-SHA256" || hp.accessKey == "" || hp.signedHeaders == "" ||
		hp.signature == "" || hp.requestTime.IsZero() {
		return fmt.Errorf("%w: signature or request time missing", ErrSignatureDoesNotMatch)
	}

	// Minimal validation of scope
	if hp.scopeDate.IsZero() || hp.scopeRegion == "" || hp.scopeService != "s3" {
		return ErrCredMalformed
	}

	// Validate time skew (+/- 5 minutes)
	if d := time.Since(hp.requestTime.UTC()); d > 5*time.Minute || d < -5*time.Minute {
		return ErrRequestNotReadyYet
	}

	return nil
}
