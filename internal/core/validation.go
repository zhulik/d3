package core

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/zhulik/d3/pkg/credentials"
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

func ValidatePartNumber(partNumber int) error {
	if partNumber < 1 || partNumber > MaxPartNumber {
		return fmt.Errorf("%w: %d", ErrInvalidPartNumber, partNumber)
	}

	return nil
}

func ValidateUploadID(uploadID string) error {
	if err := uuid.Validate(uploadID); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidUploadID, uploadID)
	}

	return nil
}

func ValidateAdminUser(user *User) error {
	if user == nil {
		return fmt.Errorf("%w: admin user is nil", ErrInvalidAdminCredentials)
	}

	if user.AccessKeyID == "" {
		return fmt.Errorf("%w: access_key_id is empty", ErrInvalidAdminCredentials)
	}

	if len(user.AccessKeyID) != credentials.AccessKeyIDLength {
		return fmt.Errorf("%w: access_key_id must be %d characters, got %d",
			ErrInvalidAdminCredentials, credentials.AccessKeyIDLength, len(user.AccessKeyID))
	}

	if user.SecretAccessKey == "" {
		return fmt.Errorf("%w: secret_access_key is empty", ErrInvalidAdminCredentials)
	}

	if len(user.SecretAccessKey) != credentials.SecretAccessKeyLength {
		return fmt.Errorf("%w: secret_access_key must be %d characters, got %d",
			ErrInvalidAdminCredentials, credentials.SecretAccessKeyLength, len(user.SecretAccessKey))
	}

	return nil
}
