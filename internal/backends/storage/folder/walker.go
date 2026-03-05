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

type WalkMultipartUploadFn func(ctx context.Context, upload *IncompleteMultipartUpload) error

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

// WalkMultipartUploads walks the multipart uploads root and calls fn for each incomplete multipart upload.
//
//nolint:funlen
func WalkMultipartUploads(
	ctx context.Context,
	bucket *Bucket,
	prefix string,
	keyMarker string,
	uploadIDMarker string,
	fn WalkMultipartUploadFn,
) error {
	multipartRoot, err := bucket.config.multipartUploadsRoot(bucket.name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(multipartRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	var startFrom *string

	if keyMarker != "" {
		if uploadIDMarker != "" {
			p := filepath.Join(multipartRoot, filepath.FromSlash(keyMarker), uploadIDMarker)
			startFrom = &p
		} else {
			p := filepath.Join(multipartRoot, filepath.FromSlash(keyMarker))
			startFrom = &p
		}
	}

	return smartio.WalkDir(ctx, multipartRoot, prefix, startFrom, func(path string) error {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(multipartRoot, path)
		if err != nil {
			return err
		}

		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) < 2 {
			return nil
		}

		upload, err := MultipartUploadFromPath(bucket, multipartRoot, path)
		if err != nil {
			return err
		}

		if upload == nil {
			return nil
		}

		if startFrom != nil && path == *startFrom {
			return nil
		}

		if keyMarker != "" && uploadIDMarker == "" && upload.Key() == keyMarker {
			return nil
		}

		err = fn(ctx, upload)
		if errors.Is(err, StopWalk) {
			return filepath.SkipAll
		}

		return err
	})
}
