package smartio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// ReadAllAndHash reads all data from r and returns the bytes, SHA256 checksum, and any error.
// The checksum is calculated during reading to maximize performance.
func ReadAllAndHash(r io.Reader) ([]byte, string, error) {
	hasher := sha256.New()
	buf := &bytes.Buffer{}

	_, err := io.Copy(io.MultiWriter(buf, hasher), r)
	if err != nil {
		return nil, "", err
	}

	return buf.Bytes(), hex.EncodeToString(hasher.Sum(nil)), nil
}
