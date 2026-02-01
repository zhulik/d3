package folder

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/pkg/smartio"
	"github.com/zhulik/d3/pkg/yaml"
)

const (
	StreamingHMACSHA256 = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
)

type BackendObjects struct {
	Cfg *core.Config

	Locker *locker.Locker

	config *Config
}

func (b *BackendObjects) Init(_ context.Context) error {
	b.config = &Config{b.Cfg}

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
	bucketPath := b.config.bucketPath(bucket)

	_, err := os.Stat(bucketPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return common.ErrBucketNotFound
		}

		return err
	}

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
	defer os.RemoveAll(uploadPath)

	uploadFile, err := os.Create(filepath.Join(uploadPath, blobFilename))
	if err != nil {
		return err
	}
	defer uploadFile.Close()

	_, sha256sum, err := smartio.Copy(ctx, uploadFile, input.Reader)
	if err != nil {
		return err
	}

	// TODO: figure out what to do with streaming uploads
	if input.Metadata.SHA256 != StreamingHMACSHA256 {
		if input.Metadata.SHA256 != sha256sum {
			return fmt.Errorf("%w: %s != %s", common.ErrObjectChecksumMismatch, input.Metadata.SHA256, sha256sum)
		}
	}

	metadata, err := objectMetadata(input, sha256sum)
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

func (b *BackendObjects) ListObjectsV2(_ context.Context, bucket string, input core.ListObjectsV2Input) ([]*types.Object, error) { //nolint:lll
	objects := []*types.Object{}

	bucketPath := b.config.bucketPath(bucket)

	// TODO: support pagination
	// TODO: when prefix is given, we should first find all directories that match the prefix and then walk through them
	err := filepath.WalkDir(bucketPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// skip files
		if !entry.IsDir() {
			return nil
		}

		if path == bucketPath {
			return nil
		}

		key, err := filepath.Rel(bucketPath, path)
		if err != nil {
			return err
		}

		key = filepath.ToSlash(key) // Normalize to forward slashes for S3 keys

		if !strings.HasPrefix(key, input.Prefix) {
			return nil
		}

		object, err := ObjectFromPath(b.config, bucket, key)
		if err != nil {
			return err
		}

		if object == nil {
			return nil
		}

		metadata, err := object.Metadata()
		if err != nil {
			return err
		}

		objects = append(objects, &types.Object{
			Key:          aws.String(key),
			LastModified: &metadata.LastModified,
			Size:         &metadata.Size,
		})

		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, common.ErrBucketNotFound
		}

		return nil, err
	}

	return objects, nil
}

func (b *BackendObjects) DeleteObjects(ctx context.Context, bucket string, quiet bool, keys ...string) ([]core.DeleteResult, error) { //nolint:lll
	results := []core.DeleteResult{}

	for _, key := range keys {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		object, err := b.getObject(bucket, key)
		if err != nil {
			results = append(results, core.DeleteResult{Key: key, Error: err})

			continue
		}

		err = object.Delete()
		if err != nil {
			results = append(results, core.DeleteResult{Key: key, Error: err})
		} else if !quiet {
			results = append(results, core.DeleteResult{Key: key, Error: nil})
		}
	}

	return results, nil
}

func (b *BackendObjects) CreateMultipartUpload(_ context.Context, _, _ string, metadata core.ObjectMetadata) (string, error) { //nolint:lll
	id, uploadPath := b.config.newMultipartUploadPath()

	err := os.MkdirAll(uploadPath, 0755)
	if err != nil {
		return "", err
	}

	err = yaml.MarshalToFile(metadata, filepath.Join(uploadPath, metadataYamlFilename))
	if err != nil {
		return "", err
	}

	return id, nil
}

func (b *BackendObjects) UploadPart(ctx context.Context, _, _ string, uploadID string, partNumber int, body io.Reader) error { //nolint:lll
	uploadPath := b.config.multipartUploadPath(uploadID)
	path := filepath.Join(uploadPath, fmt.Sprintf("part-%d", partNumber))

	_, cancel, err := b.Locker.Lock(ctx, path)
	if err != nil {
		return err
	}
	defer cancel()

	// TODO: this behavior should depend on the passed details
	if _, err := os.Stat(path); err == nil {
		return common.ErrObjectAlreadyExists
	}

	uploadFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer uploadFile.Close()

	_, _, err = smartio.Copy(ctx, uploadFile, body)
	if err != nil {
		return err
	}

	return nil
}

func (b *BackendObjects) CompleteMultipartUpload(ctx context.Context, bucket, key string, uploadID string, parts []core.CompletePart) error { //nolint:lll,funlen
	slices.SortFunc(parts, func(a, b core.CompletePart) int {
		return a.PartNumber - b.PartNumber
	})

	// validate that all parts are present
	for _, part := range parts {
		uploadPath := b.config.multipartUploadPath(uploadID)

		path := filepath.Join(uploadPath, fmt.Sprintf("part-%d", part.PartNumber))
		if _, err := os.Stat(path); err != nil {
			return err
		}
	}

	uploadPath := b.config.multipartUploadPath(uploadID)

	blobFile, err := os.Create(filepath.Join(uploadPath, blobFilename))
	if err != nil {
		return err
	}
	defer blobFile.Close()

	for _, part := range parts {
		path := filepath.Join(uploadPath, fmt.Sprintf("part-%d", part.PartNumber))

		partFile, err := os.Open(path)
		if err != nil {
			return err
		}

		_, _, err = smartio.Copy(ctx, blobFile, partFile)
		if err != nil {
			partFile.Close()

			return err
		}

		partFile.Close()
	}

	files, err := os.ReadDir(uploadPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "part-") {
			err := os.Remove(filepath.Join(uploadPath, file.Name()))
			if err != nil {
				return err
			}
		}
	}

	blobFileStat, err := blobFile.Stat()
	if err != nil {
		return err
	}

	metadata, err := yaml.UnmarshalFromFile[core.ObjectMetadata](filepath.Join(uploadPath, metadataYamlFilename))
	if err != nil {
		return err
	}

	metadata.Size = blobFileStat.Size()

	err = yaml.MarshalToFile(metadata, filepath.Join(uploadPath, metadataYamlFilename))
	if err != nil {
		return err
	}

	err = os.Rename(uploadPath, b.config.objectPath(bucket, key))
	if err != nil {
		return err
	}

	return nil
}

func (b *BackendObjects) AbortMultipartUpload(_ context.Context, _, _ string, uploadID string) error {
	uploadPath := b.config.multipartUploadPath(uploadID)

	return os.RemoveAll(uploadPath)
}

func (b *BackendObjects) getObject(bucket, key string) (*Object, error) {
	object, err := ObjectFromPath(b.config, bucket, key)
	if err != nil {
		return nil, err
	}

	if object == nil {
		return nil, common.ErrObjectNotFound
	}

	return object, nil
}

func objectMetadata(input core.PutObjectInput, sha256 string) (core.ObjectMetadata, error) {
	rawSha256, err := hex.DecodeString(sha256)
	if err != nil {
		return core.ObjectMetadata{}, err
	}

	return core.ObjectMetadata{
		ContentType:  input.Metadata.ContentType,
		Tags:         input.Metadata.Tags,
		SHA256:       sha256,
		SHA256Base64: base64.StdEncoding.EncodeToString(rawSha256),
		Size:         input.Metadata.Size,
		LastModified: time.Now(),
		Meta:         input.Metadata.Meta,
	}, nil
}
