package folder

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iter"
)

type BackendBuckets struct {
	Cfg *core.Config

	config *config
}

func (b *BackendBuckets) Init(_ context.Context) error {
	b.config = &config{b.Cfg}
	return nil
}

func (b *BackendBuckets) ListBuckets(_ context.Context) ([]*types.Bucket, error) {
	entries, err := os.ReadDir(b.config.FolderBackendPath)
	if err != nil {
		return nil, err
	}

	return iter.ErrFilterMap(entries, func(entry os.DirEntry) (*types.Bucket, bool, error) {
		if !entry.IsDir() {
			return nil, false, nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil, false, err
		}
		return &types.Bucket{
			Name:         aws.String(entry.Name()),
			CreationDate: aws.Time(info.ModTime()),
			BucketRegion: aws.String("local"),
			BucketArn:    aws.String("arn:aws:s3:::" + entry.Name()),
		}, true, nil
	})
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

	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return common.ErrBucketNotFound
		}
		return err
	}

	if len(entries) > 0 {
		return common.ErrBucketNotEmpty
	}
	return os.Remove(path)
}

func (b *BackendBuckets) HeadBucket(_ context.Context, name string) error {
	path := b.config.bucketPath(name)
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return common.ErrBucketNotFound
		}
		return err
	}
	return nil
}
