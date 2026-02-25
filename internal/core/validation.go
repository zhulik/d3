package core

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

var bucketNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]{1,61}[a-z0-9]$`)

func ValidateBucketName(name string) error {
	if !bucketNameRegexp.MatchString(name) {
		return fmt.Errorf("%w: %q", ErrInvalidBucketName, name)
	}

	if strings.Contains(name, "..") {
		return fmt.Errorf("%w: %q contains '..'", ErrInvalidBucketName, name)
	}

	return nil
}

func ValidateObjectKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: key is empty", ErrInvalidObjectKey)
	}

	if slices.Contains(strings.Split(key, "/"), "..") {
		return fmt.Errorf("%w: %q contains '..' segment", ErrInvalidObjectKey, key)
	}

	return nil
}

func ValidateUploadID(uploadID string) error {
	if err := uuid.Validate(uploadID); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidUploadID, uploadID)
	}

	return nil
}
