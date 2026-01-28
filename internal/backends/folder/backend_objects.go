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
	Cfg *core.Config

	Locker             *locker.Locker
	MetadataRepository *MetadataRepository

	config *config
}

func (b *BackendObjects) Init(_ context.Context) error {
	b.config = &config{b.Cfg}
	return nil
}

func (b *BackendObjects) HeadObject(ctx context.Context, bucket, key string) (*core.ObjectMetadata, error) {
	return b.MetadataRepository.Get(ctx, bucket, key)
}

func (b *BackendObjects) PutObject(ctx context.Context, bucket, key string, input core.PutObjectInput) error {
	path := b.config.objectPath(bucket, key)

	_, cancel, err := b.Locker.Lock(ctx, path)
	if err != nil {
		return err
	}
	defer cancel()

	// TODO: this behavior should depend on the passed details
	if _, err := os.Stat(path); err == nil {
		return common.ErrObjectAlreadyExists
	}

	uploadPath := b.config.newUploadPath()
	err = os.MkdirAll(uploadPath, 0755)
	if err != nil {
		return err
	}
	defer os.RemoveAll(uploadPath) //nolint:errcheck

	uploadFile, err := os.Create(filepath.Join(uploadPath, blobFilename))
	if err != nil {
		return err
	}
	defer uploadFile.Close() //nolint:errcheck

	_, sha256sum, err := smartio.Copy(ctx, uploadFile, input.Reader)
	if err != nil {
		return err
	}

	if input.Metadata.SHA256 != sha256sum {
		return common.ErrObjectChecksumMismatch
	}

	rawSha256, err := hex.DecodeString(sha256sum)
	if err != nil {
		return err
	}

	metadata := core.ObjectMetadata{
		ContentType:  input.Metadata.ContentType,
		Tags:         input.Metadata.Tags,
		SHA256:       sha256sum,
		SHA256Base64: base64.StdEncoding.EncodeToString(rawSha256),
		Size:         input.Metadata.Size,
		LastModified: time.Now(),
		Meta:         input.Metadata.Meta,
	}

	err = b.MetadataRepository.SaveTmp(ctx, uploadPath, &metadata)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	err = os.Rename(uploadPath, path)
	if err != nil {
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
	path := b.config.objectPath(bucket, key)

	f, err := os.Open(filepath.Join(path, blobFilename))
	if err != nil {
		return nil, err
	}

	metadata, err := b.MetadataRepository.Get(ctx, bucket, key)
	if err != nil {
		f.Close() //nolint:errcheck
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
	entries, err := os.ReadDir(b.config.objectPath(bucket, prefix))
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

func (b *BackendObjects) DeleteObject(_ context.Context, bucket, key string) error {
	path := b.config.objectPath(bucket, key)

	err := os.RemoveAll(path)
	if err != nil {
		return err
	}

	return nil
}
