package folder

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
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
	"github.com/zhulik/d3/pkg/yaml"
)

type BackendObjects struct {
	Cfg *core.Config

	Locker *locker.Locker

	config *config
}

func (b *BackendObjects) Init(_ context.Context) error {
	b.config = &config{b.Cfg}
	return nil
}

func (b *BackendObjects) HeadObject(_ context.Context, bucket, key string) (*core.ObjectMetadata, error) {
	object, err := b.getObject(bucket, key)
	if err != nil {
		return nil, err
	}
	return object.Metadata()
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
		return fmt.Errorf("%w: %s != %s", common.ErrObjectChecksumMismatch, input.Metadata.SHA256, sha256sum)
	}

	metadata, err := objectMetadata(input)
	if err != nil {
		return err
	}

	err = yaml.MarshalToFile(metadata, filepath.Join(uploadPath, metadataYamlFilename))
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

func (b *BackendObjects) GetObjectTagging(_ context.Context, bucket, key string) (map[string]string, error) {
	object, err := b.getObject(bucket, key)
	if err != nil {
		return nil, err
	}
	metadata, err := object.Metadata()
	if err != nil {
		return nil, err
	}
	return metadata.Tags, nil
}

func (b *BackendObjects) GetObject(_ context.Context, bucket, key string) (*core.ObjectContent, error) {
	object, err := b.getObject(bucket, key)
	if err != nil {
		return nil, err
	}
	metadata, err := object.Metadata()
	if err != nil {
		return nil, err
	}

	return &core.ObjectContent{
		Reader:   object,
		Metadata: metadata,
	}, nil
}

func (b *BackendObjects) ListObjectsV2(_ context.Context, bucket, prefix string) ([]*types.Object, error) {
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
	object, err := b.getObject(bucket, key)
	if err != nil {
		return err
	}

	return object.Delete()
}

func (b *BackendObjects) getObject(bucket, key string) (*Object, error) {
	path := b.config.objectPath(bucket, key)

	object, err := ObjectFromPath(path)
	if err != nil {
		return nil, err
	}
	if object == nil {
		return nil, common.ErrObjectNotFound
	}
	return object, nil
}

func objectMetadata(input core.PutObjectInput) (core.ObjectMetadata, error) {
	rawSha256, err := hex.DecodeString(input.Metadata.SHA256)
	if err != nil {
		return core.ObjectMetadata{}, err
	}

	return core.ObjectMetadata{
		ContentType:  input.Metadata.ContentType,
		Tags:         input.Metadata.Tags,
		SHA256:       input.Metadata.SHA256,
		SHA256Base64: base64.StdEncoding.EncodeToString(rawSha256),
		Size:         input.Metadata.Size,
		LastModified: time.Now(),
		Meta:         input.Metadata.Meta,
	}, nil
}
