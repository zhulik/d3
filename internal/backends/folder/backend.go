package folder

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/pkg/iter"
)

type Backend struct {
	Config *core.Config
	Locker *locker.Locker
}

func (b *Backend) ListBuckets(ctx context.Context) ([]*types.Bucket, error) {
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

func (b *Backend) CreateBucket(ctx context.Context, name string) error {
	path := filepath.Join(b.Config.FolderBackendPath, name)
	return os.Mkdir(path, 0755)
}

func (b *Backend) DeleteBucket(ctx context.Context, name string) error {
	path := filepath.Join(b.Config.FolderBackendPath, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return common.ErrBucketNotFound
	}
	return os.RemoveAll(path)
}

func (b *Backend) HeadBucket(ctx context.Context, name string) error {
	path := filepath.Join(b.Config.FolderBackendPath, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return common.ErrBucketNotFound
	}
	return nil
}

func (b *Backend) HeadObject(ctx context.Context, bucket, key string) (*core.HeadObjectResult, error) {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)
	fileInfo, err := os.Stat(path)

	if os.IsNotExist(err) {
		return nil, common.ErrObjectNotFound
	}

	return &core.HeadObjectResult{
		LastModified:  fileInfo.ModTime(),
		ContentLength: fileInfo.Size(),
	}, nil
}

func (b *Backend) PutObject(ctx context.Context, bucket, key string, reader io.Reader) error {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	ctx, cancel, err := b.Locker.Lock(ctx, path)
	if err != nil {
		return err
	}
	defer cancel()

	if _, err := os.Stat(path); err == nil {
		return common.ErrObjectAlreadyExists
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	_, err = io.Copy(f, reader) // TODO: make it cancellable
	if err != nil {
		f.Close()
		os.Remove(path)
	}

	return err
}

func (b *Backend) GetObject(ctx context.Context, bucket, key string) (*core.ObjectContent, error) {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)
	fileinfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &core.ObjectContent{
		ReadCloser:   f,
		LastModified: fileinfo.ModTime(),
		Size:         fileinfo.Size(),
	}, nil
}

func (b *Backend) ListObjects(ctx context.Context, bucket, prefix string) ([]*types.Object, error) {
	entries, err := os.ReadDir(filepath.Join(b.Config.FolderBackendPath, bucket, prefix))
	if err != nil {
		return nil, err
	}
	return iter.ErrMap(entries, func(entry os.DirEntry) (*types.Object, error) {
		fileInfo, err := entry.Info()
		if err != nil {
			return nil, err
		}
		return &types.Object{
			Key:          aws.String("/" + filepath.Join(prefix, entry.Name())),
			LastModified: aws.Time(fileInfo.ModTime()),
			Size:         aws.Int64(fileInfo.Size()),
		}, nil
	})
}

func (b *Backend) DeleteObject(ctx context.Context, bucket, key string) error {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)
	return os.Remove(path)
}
