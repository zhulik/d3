package folder

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/smartio"
)

var StopWalk = errors.New("stop walk") //nolint:revive,errname,gochecknoglobals,staticcheck

type WalkFn func(ctx context.Context, object core.Object) error

// WalkBucket walks the bucket and calls the given function for each object in the bucket.
// TODO: support pagination
// TODO: when prefix is given, we should first find all directories that match the prefix and then walk through them.
func WalkBucket(ctx context.Context, bucket *Bucket, prefix string, fn WalkFn) error {
	return smartio.WalkDir(ctx, bucket.rootPath(), prefix, nil, func(path string) error {
		key := strings.TrimPrefix(path, bucket.rootPath())
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
