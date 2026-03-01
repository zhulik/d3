package smartio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

const defaultCopyBufferSize = 32 * 1024

// copyContext copies from src to dst until EOF or context cancellation.
// It checks for context cancellation before each read.
func copyContext(ctx context.Context, dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if buf == nil {
		buf = make([]byte, defaultCopyBufferSize)
	}

	var written int64

	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])

			written += int64(nw)
			if ew != nil {
				return written, ew
			}

			if nr != nw {
				return written, io.ErrShortWrite
			}
		}

		if er != nil {
			if er != io.EOF {
				return written, er
			}

			return written, nil
		}
	}
}

// CopyAll copies data from multiple readers into dst and returns the number of bytes written,
// the SHA256 checksum of the concatenated contents, and any error.
// It respects context cancellation and returns ctx.Err() when the context is done.
func CopyAll(ctx context.Context, dst io.Writer, src ...io.Reader) (int64, string, error) {
	hasher := sha256.New()
	multiWriter := io.MultiWriter(dst, hasher)

	written, err := copyContext(ctx, multiWriter, io.MultiReader(src...), nil)
	if err != nil {
		return 0, "", err
	}

	return written, hex.EncodeToString(hasher.Sum(nil)), nil
}

// Copy copies the contents of src to dst and returns the number of bytes written and
// the SHA256 checksum of the contents.
// It respects context cancellation and returns ctx.Err() when the context is done.
func Copy(ctx context.Context, dst io.Writer, src io.Reader) (int64, string, error) {
	return CopyAll(ctx, dst, src)
}
