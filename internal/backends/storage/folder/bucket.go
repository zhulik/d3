package folder

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/smartio"
	"github.com/zhulik/d3/pkg/yaml"
)

type Bucket struct {
	name         string
	creationDate time.Time
	config       *Config

	Locker core.Locker
}

func (b *Bucket) Name() string {
	return b.name
}

func (b *Bucket) ARN() string {
	return "arn:aws:s3:::" + b.Name()
}

func (b *Bucket) Region() string {
	return "local"
}

func (b *Bucket) CreationDate() time.Time {
	return b.creationDate
}

func (b *Bucket) HeadObject(_ context.Context, key string) (core.Object, error) {
	object, err := b.getObject(key)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (b *Bucket) PutObject(ctx context.Context, key string, input core.PutObjectInput) error {
	path := b.config.objectPath(b.name, key)

	_, cancel, err := b.Locker.Lock(ctx, path)
	if err != nil {
		return err
	}
	defer cancel()

	// TODO: this behavior should depend on the passed details
	if _, err := os.Stat(path); err == nil {
		return core.ErrObjectAlreadyExists
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

	if input.Metadata.SHA256 != sha256sum {
		return fmt.Errorf("%w: %s != %s", core.ErrObjectChecksumMismatch, input.Metadata.SHA256, sha256sum)
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

func (b *Bucket) GetObject(_ context.Context, key string) (core.Object, error) {
	object, err := b.getObject(key)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (b *Bucket) ListObjectsV2(ctx context.Context, input core.ListObjectsV2Input) (*core.ListV2Result, error) {
	objects := []core.Object{}

	isTruncated := false

	var continuationToken *string

	var nextKey *string

	if input.ContinuationToken != "" {
		decodedKey, err := base64.StdEncoding.DecodeString(input.ContinuationToken)
		if err != nil {
			return nil, err
		}

		nextKey = lo.ToPtr(string(decodedKey))
	}

	err := WalkBucket(ctx, b, input.Prefix, nextKey, func(_ context.Context, object core.Object) error {
		if len(objects) >= input.MaxKeys {
			// there is at least one more object after the last one, so we consider the list truncated
			isTruncated = true

			continuationToken = lo.ToPtr(base64.StdEncoding.EncodeToString([]byte(object.Key())))

			return StopWalk
		}

		objects = append(objects, object)

		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, err
	}

	return &core.ListV2Result{
		Objects:           objects,
		ContinuationToken: continuationToken,
		IsTruncated:       isTruncated,
	}, nil
}

func (b *Bucket) DeleteObjects(ctx context.Context, quiet bool, keys ...string) ([]core.DeleteResult, error) { //nolint:lll
	results := []core.DeleteResult{}

	for _, key := range keys {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		object, err := b.getObject(key)
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

func (b *Bucket) CreateMultipartUpload(_ context.Context, _ string, metadata core.ObjectMetadata) (string, error) { //nolint:lll
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

func (b *Bucket) UploadPart(ctx context.Context, _ string, uploadID string, partNumber int, body io.Reader) error { //nolint:lll
	uploadPath := b.config.multipartUploadPath(uploadID)
	path := filepath.Join(uploadPath, fmt.Sprintf("part-%d", partNumber))

	_, cancel, err := b.Locker.Lock(ctx, path)
	if err != nil {
		return err
	}
	defer cancel()

	// TODO: this behavior should depend on the passed details
	if _, err := os.Stat(path); err == nil {
		return core.ErrObjectAlreadyExists
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

func (b *Bucket) CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []core.CompletePart) error { //nolint:lll,funlen
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

	err = os.Rename(uploadPath, b.config.objectPath(b.name, key))
	if err != nil {
		return err
	}

	return nil
}

func (b *Bucket) AbortMultipartUpload(_ context.Context, _ string, uploadID string) error {
	uploadPath := b.config.multipartUploadPath(uploadID)

	return os.RemoveAll(uploadPath)
}

func (b *Bucket) getObject(key string) (*Object, error) {
	object, err := ObjectFromPath(b, key)
	if err != nil {
		return nil, err
	}

	if object == nil {
		return nil, core.ErrObjectNotFound
	}

	return object, nil
}

func (b *Bucket) rootPath() string {
	return b.config.bucketPath(b.name)
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
