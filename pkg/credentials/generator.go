package credentials

import (
	"crypto/rand"
)

const (
	AccessKeyIDPrefix       = "AKIA"
	AccessKeyIDLength       = 20
	AccessKeyIDSuffixLength = AccessKeyIDLength - len(AccessKeyIDPrefix)
	SecretAccessKeyLength   = 40

	AccessKeyIDCharset     = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	SecretAccessKeyCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
)

// GenerateCredentials generates both access key ID and secret access key.
func GenerateCredentials() (string, string) {
	return AccessKeyIDPrefix + generateRandomString(AccessKeyIDSuffixLength, AccessKeyIDCharset),
		generateRandomString(SecretAccessKeyLength, SecretAccessKeyCharset)
}

// generateRandomString generates a random string of the specified length
// using characters from the provided charset.
func generateRandomString(length int, charset string) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	result := make([]byte, length)
	for i := range length {
		result[i] = charset[bytes[i]%byte(len(charset))] //nolint:gosec
	}

	return string(result)
}
