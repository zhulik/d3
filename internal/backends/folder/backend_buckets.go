package folder

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iter"
)

type BackendBuckets struct {
	Config *core.Config
}

func (b *BackendBuckets) ListBuckets(_ context.Context) ([]*types.Bucket, error) {
	entries, err := os.ReadDir(b.Config.FolderBackendPath)
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
	path := filepath.Join(b.Config.FolderBackendPath, name)
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
	path := filepath.Join(b.Config.FolderBackendPath, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return common.ErrBucketNotFound
	}
	return os.RemoveAll(path)
}

func (b *BackendBuckets) HeadBucket(_ context.Context, name string) error {
	path := filepath.Join(b.Config.FolderBackendPath, name)
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return common.ErrBucketNotFound
		}
		return err
	}
	return nil
}
