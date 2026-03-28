package atomicwriter

import (
	"context"
	"os"
	"path/filepath"
)

type ContentMapFunc func(ctx context.Context, content []byte) ([]byte, error)

//go:generate go tool mockery
type Locker interface {
	Lock(ctx context.Context, key string) (context.Context, context.CancelFunc, error)
}

type AtomicWriter struct {
	Locker  Locker
	tmpPath string
}

func New(locker Locker, tmpPath string) *AtomicWriter {
	return &AtomicWriter{
		Locker:  locker,
		tmpPath: tmpPath,
	}
}

// ReadWrite locks the file with the given filename, reads its content, applies the contentMap function to get
// the new content, and writes the new content back to the file atomically. The lock is released after the operation
// is complete. If the file does not exist, it will be treated as an empty file.
// The caller is responsible for ensuring that the directory of the file exists.
func (w *AtomicWriter) ReadWrite(ctx context.Context, filename string, contentMap ContentMapFunc) error { //nolint:funlen,lll
	ctx, cancel, err := w.Locker.Lock(ctx, filename)
	if err != nil {
		return err
	}
	defer cancel()

	// We can't create the file in a non-existent directory, it's caller responsibility to make sure the directory exists
	_, err = os.Stat(filepath.Dir(filename))
	if err != nil {
		return err
	}

	// Capture original file permissions
	var originalPerm os.FileMode

	fileInfo, err := os.Stat(filename)
	if err != nil {
		// if the file does not exist, we use default permission
		if os.IsNotExist(err) {
			originalPerm = 0644
		} else {
			return err
		}
	} else {
		originalPerm = fileInfo.Mode()
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		// if the file does not exist, we pretend we have an empty file
		if os.IsNotExist(err) {
			content = []byte{}
		} else {
			return err
		}
	}

	newContent, err := contentMap(ctx, content)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(w.tmpPath, "atomic-writer-*.tmp")
	if err != nil {
		return err
	}
	// For simplicity, we try to remove the temp file anyways. If the rename succeeds,
	// Remove will return an error, but we ignore it
	defer os.Remove(tempFile.Name())

	defer tempFile.Close()

	_, err = tempFile.Write(newContent)
	if err != nil {
		return err
	}

	// Only perform rename if we still have the lock and the context is not cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	err = tempFile.Sync()
	if err != nil {
		return err
	}

	// Set the temp file permissions to match the original file
	err = tempFile.Chmod(originalPerm)
	if err != nil {
		return err
	}

	err = os.Rename(tempFile.Name(), filename)
	if err != nil {
		return err
	}

	return nil
}
