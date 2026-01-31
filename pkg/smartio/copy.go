package smartio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// Copy copies the contents of src to dst and returns the number of bytes written and
// the SHA256 checksum of the contents.
// TODO: respect context cancellation.
func Copy(_ context.Context, dst io.Writer, src io.Reader) (int64, string, error) {
	hasher := sha256.New()

	written, err := io.Copy(io.MultiWriter(dst, hasher), src)
	if err != nil {
		return 0, "", err
	}

	return written, hex.EncodeToString(hasher.Sum(nil)), nil
}
