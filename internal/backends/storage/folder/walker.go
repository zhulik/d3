package folder

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/smartio"
)

var StopWalk = errors.New("stop walk") //nolint:revive,errname,gochecknoglobals,staticcheck

type WalkFn func(ctx context.Context, object core.Object) error

// WalkBucket walks the bucket and calls the given function for each object in the bucket.
func WalkBucket(ctx context.Context, bucket *Bucket, prefix string, nextKey *string, fn WalkFn) error {
	bucketRoot, err := bucket.rootPath()
	if err != nil {
		return err
	}

	objectsRoot := filepath.Join(bucketRoot, objectsFolder)

	if _, err := os.Stat(objectsRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	var startFrom *string

	if nextKey != nil {
		fullPath := filepath.Join(objectsRoot, filepath.FromSlash(*nextKey))
		startFrom = &fullPath
	}

	return smartio.WalkDir(ctx, objectsRoot, prefix, startFrom, func(path string) error {
		key := strings.TrimPrefix(path, objectsRoot)
		key = strings.TrimPrefix(key, "/")

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
