package folder

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/zhulik/d3/internal/core"
)

var StopWalk = errors.New("stop walk") //nolint:revive,errname,gochecknoglobals,staticcheck

type WalkFn func(ctx context.Context, object core.Object) error

// WalkBucket walks the bucket and calls the given function for each object in the bucket.
// TODO: support pagination
// TODO: when prefix is given, we should first find all directories that match the prefix and then walk through them.
func WalkBucket(ctx context.Context, bucket *Bucket, prefix string, fn WalkFn) error {
	return filepath.WalkDir(bucket.rootPath(), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if !entry.IsDir() {
			return nil
		}

		if path == bucket.rootPath() {
			return nil
		}

		key, err := filepath.Rel(bucket.rootPath(), path)
		if err != nil {
			return err
		}

		key = filepath.ToSlash(key) // Normalize to forward slashes for S3 keys

		if !strings.HasPrefix(key, prefix) {
			return nil
		}

		object, err := ObjectFromPath(bucket, key)
		if err != nil {
			return err
		}

		if object == nil {
			return nil
		}

		err = fn(ctx, object)
		if errors.Is(err, StopWalk) {
			return filepath.SkipAll
		}

		return err
	})
}
