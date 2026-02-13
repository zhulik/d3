package sigv4

import (
	"fmt"
	"time"
)

type AuthHeaderParameters struct {
	algo          string
	accessKey     string
	scopeDate     string
	scopeRegion   string
	scopeService  string
	signedHeaders string
	signature     string
	requestTime   string
	hashedPayload string
}

func (hp *AuthHeaderParameters) Validate() error {
	if hp.algo != "AWS4-HMAC-SHA256" || hp.accessKey == "" || hp.signedHeaders == "" ||
		hp.signature == "" || hp.requestTime == "" {
		return fmt.Errorf("%w: signature or request time missing", ErrSignatureDoesNotMatch)
	}

	// Minimal validation of scope
	if hp.scopeDate == "" || hp.scopeRegion == "" || hp.scopeService != "s3" {
		return ErrCredMalformed
	}

	// Validate time skew (+/- 5 minutes)
	tReq, err := time.Parse("20060102T150405Z", hp.requestTime)
	if err != nil {
		return ErrMalformedPresignedDate
	}

	if d := time.Since(tReq.UTC()); d > 5*time.Minute || d < -5*time.Minute {
		return ErrRequestNotReadyYet
	}

	return nil
}
