package folder

import (
	"context"
	"errors"
	"os"
	"syscall"

	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/pkg/iter"
)

type BackendBuckets struct {
	Cfg *core.Config

	config *Config

	Locker *locker.Locker
}

func (b *BackendBuckets) Init(_ context.Context) error {
	b.config = &Config{b.Cfg}

	return nil
}

func (b *BackendBuckets) ListBuckets(_ context.Context) ([]core.Bucket, error) {
	entries, err := os.ReadDir(b.config.bucketsPath())
	if err != nil {
		return nil, err
	}

	return iter.ErrMap(entries, b.dirEntryToBucket)
}

func (b *BackendBuckets) CreateBucket(_ context.Context, name string) error {
	path := b.config.bucketPath(name)

	err := os.Mkdir(path, 0755)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return common.ErrBucketAlreadyExists
		}

		return err
	}

	return nil
}

func (b *BackendBuckets) DeleteBucket(_ context.Context, name string) error {
	path := b.config.bucketPath(name)

	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return common.ErrBucketNotFound
		}

		var pathError *os.PathError
		if errors.As(err, &pathError) {
			if errors.Is(pathError.Err, syscall.ENOTEMPTY) {
				return common.ErrBucketNotEmpty
			}
		}

		return err
	}

	return nil
}

func (b *BackendBuckets) HeadBucket(_ context.Context, name string) (core.Bucket, error) {
	path := b.config.bucketPath(name)

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, common.ErrBucketNotFound
		}

		return nil, err
	}

	return &Bucket{
		name:         name,
		creationDate: info.ModTime(), // TODO: use the actual creation date
		config:       b.config,
		Locker:       b.Locker,
	}, nil
}

func (b *BackendBuckets) dirEntryToBucket(entry os.DirEntry) (core.Bucket, error) {
	info, err := entry.Info()
	if err != nil {
		return nil, err
	}

	return &Bucket{
		name:         entry.Name(),
		creationDate: info.ModTime(), // TODO: use the actual creation date
		config:       b.config,
		Locker:       b.Locker,
	}, nil
}
