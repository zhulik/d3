package folder

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/pkg/iter"
	"github.com/zhulik/d3/pkg/smartio"
)

type Backend struct {
	Logger *slog.Logger

	Config             *core.Config
	Locker             *locker.Locker
	MetadataRepository *MetadataRepository
}

func (b *Backend) ListBuckets(_ context.Context) ([]*types.Bucket, error) {
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

func (b *Backend) CreateBucket(_ context.Context, name string) error {
	path := filepath.Join(b.Config.FolderBackendPath, name)
	return os.Mkdir(path, 0755)
}

func (b *Backend) DeleteBucket(_ context.Context, name string) error {
	path := filepath.Join(b.Config.FolderBackendPath, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return common.ErrBucketNotFound
	}
	return os.RemoveAll(path)
}

func (b *Backend) HeadBucket(_ context.Context, name string) error {
	path := filepath.Join(b.Config.FolderBackendPath, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return common.ErrBucketNotFound
	}
	return nil
}

func (b *Backend) HeadObject(ctx context.Context, bucket, key string) (*core.ObjectMetadata, error) {
	return b.MetadataRepository.Get(ctx, bucket, key)
}

func (b *Backend) PutObject(ctx context.Context, bucket, key string, input core.PutObjectInput) error {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	_, cancel, err := b.Locker.Lock(ctx, path)
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

	_, sha256sum, err := smartio.Copy(ctx, f, input.Reader)
	if err != nil {
		f.Close() //nolint:errcheck
		rmErr := os.Remove(path)
		err = errors.Join(err, rmErr)
		return err
	}

	if input.Metadata.SHA256 != sha256sum {
		return common.ErrObjectChecksumMismatch
	}

	err = b.MetadataRepository.Save(ctx, bucket, key, &core.ObjectMetadata{
		ContentType:  input.Metadata.ContentType,
		Tags:         input.Metadata.Tags,
		SHA256:       sha256sum,
		Size:         input.Metadata.Size,
		LastModified: time.Now(),
		Meta:         input.Metadata.Meta,
	})
	if err != nil {
		f.Close() //nolint:errcheck
		rmErr := os.Remove(path)
		err = errors.Join(err, rmErr)
		return err
	}

	return nil
}

func (b *Backend) GetObject(ctx context.Context, bucket, key string) (*core.ObjectContent, error) {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	metadata, err := b.MetadataRepository.Get(ctx, bucket, key)
	if err != nil {
		return nil, err
	}

	rawSha256, err := hex.DecodeString(metadata.SHA256)
	if err != nil {
		return nil, err
	}
	sha256Base64 := base64.StdEncoding.EncodeToString(rawSha256)

	return &core.ObjectContent{
		ReadCloser: f,
		ObjectMetadata: core.ObjectMetadata{
			LastModified: metadata.LastModified,
			Size:         metadata.Size,
			ContentType:  metadata.ContentType,
			Tags:         metadata.Tags,
			SHA256:       metadata.SHA256,
			SHA256Base64: sha256Base64,
		},
	}, nil
}

func (b *Backend) ListObjects(_ context.Context, bucket, prefix string) ([]*types.Object, error) {
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

func (b *Backend) DeleteObject(_ context.Context, bucket, key string) error {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)
	return os.Remove(path)
}
