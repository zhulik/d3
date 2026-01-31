package folder

import (
	"context"
	"os"
	"path/filepath"

	"github.com/zhulik/d3/internal/locker"
)

type ContentMapFunc func(ctx context.Context, content []byte) ([]byte, error)

type AtomicWriter struct {
	Locker *locker.Locker
	Config *Config
}

func (w *AtomicWriter) ReadWrite(ctx context.Context, filename string, contentMap ContentMapFunc) error {
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

	tempFile, err := os.CreateTemp(w.Config.tmpPath(), "d3-*.tmp")
	if err != nil {
		return err
	}
	// For simplicity, we try to remove the temp file anyways. If the rename succeeds,
	// Remove will return an error, but we ignore it
	defer os.Remove(tempFile.Name()) //nolint:errcheck

	defer tempFile.Close() //nolint:errcheck

	_, err = tempFile.Write(newContent)
	if err != nil {
		return err
	}

	err = tempFile.Sync()
	if err != nil {
		return err
	}

	// Only perform rename if we still have the lock and the context is not cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	err = os.Rename(tempFile.Name(), filename)
	if err != nil {
		return err
	}

	return nil
}
