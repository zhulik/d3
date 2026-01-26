package folder

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
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

type BackendObjects struct {
	Config             *core.Config
	Locker             *locker.Locker
	MetadataRepository *MetadataRepository
}

func (b *BackendObjects) HeadObject(ctx context.Context, bucket, key string) (*core.ObjectMetadata, error) {
	metadata, err := b.MetadataRepository.Get(ctx, bucket, key)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func (b *BackendObjects) PutObject(ctx context.Context, bucket, key string, input core.PutObjectInput) error {
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

	rawSha256, err := hex.DecodeString(sha256sum)
	if err != nil {
		return err
	}

	err = b.MetadataRepository.Save(ctx, bucket, key, &core.ObjectMetadata{
		ContentType:  input.Metadata.ContentType,
		Tags:         input.Metadata.Tags,
		SHA256:       sha256sum,
		SHA256Base64: base64.StdEncoding.EncodeToString(rawSha256),
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

func (b *BackendObjects) GetObjectTagging(ctx context.Context, bucket, key string) (map[string]string, error) {
	metadata, err := b.MetadataRepository.Get(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	return metadata.Tags, nil
}

func (b *BackendObjects) GetObject(ctx context.Context, bucket, key string) (*core.ObjectContent, error) {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	metadata, err := b.MetadataRepository.Get(ctx, bucket, key)
	if err != nil {
		return nil, err
	}

	return &core.ObjectContent{
		Reader:   f,
		Metadata: metadata,
	}, nil
}

func (b *BackendObjects) ListObjects(_ context.Context, bucket, prefix string) ([]*types.Object, error) {
	if prefix == "" {
		prefix = "/"
	}
	entries, err := os.ReadDir(filepath.Join(b.Config.FolderBackendPath, bucket, prefix))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, common.ErrBucketNotFound
		}
		return nil, err
	}
	return iter.ErrMap(entries, func(entry os.DirEntry) (*types.Object, error) {
		fileInfo, err := entry.Info()
		if err != nil {
			return nil, err
		}
		return &types.Object{
			Key:          aws.String(filepath.Join(prefix, entry.Name())),
			LastModified: aws.Time(fileInfo.ModTime()),
			Size:         aws.Int64(fileInfo.Size()),
		}, nil
	})
}

func (b *BackendObjects) DeleteObject(ctx context.Context, bucket, key string) error {
	path := filepath.Join(b.Config.FolderBackendPath, bucket, key)

	err := os.Remove(path)
	if err != nil {
		return err
	}

	err = b.MetadataRepository.Delete(ctx, bucket, key)
	if err != nil {
		return err
	}
	return nil
}
