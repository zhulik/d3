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
	"github.com/zhulik/d3/internal/apis/s3"
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
	return b.getObject(key)
}

func (b *Bucket) PutObject(ctx context.Context, key string, input core.PutObjectInput) error { //nolint:funlen
	path, err := b.config.objectPath(b.name, key)
	if err != nil {
		return err
	}

	_, cancel, err := b.Locker.Lock(ctx, path)
	if err != nil {
		return err
	}
	defer cancel()

	if err := rejectSymlinkInPath(path); err != nil {
		return err
	}

	if _, err := os.Lstat(path); err == nil {
		if input.IfNoneMatch {
			return core.ErrPreconditionFailed
		}

		existing, err := ObjectFromPath(b, key)
		if err != nil {
			return err
		}

		if err := existing.Delete(); err != nil {
			return err
		}
	}

	uploadPath := b.config.newUploadPath()

	if err := mkdirAllNoFollow(uploadPath, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(uploadPath)

	uploadFile, err := createFileNoFollow(filepath.Join(uploadPath, blobFilename), 0644)
	if err != nil {
		return err
	}
	defer uploadFile.Close()

	actualSize, sha256sum, err := smartio.Copy(ctx, uploadFile, input.Reader)
	if err != nil {
		return err
	}

	if input.Metadata.SHA256 == s3.StreamingHMACSHA256 {
		input.Metadata.SHA256 = sha256sum
	}

	if input.Metadata.SHA256 != sha256sum {
		return fmt.Errorf("%w: %s != %s", core.ErrObjectChecksumMismatch, input.Metadata.SHA256, sha256sum)
	}

	input.Metadata.Size = actualSize

	metadata, err := objectMetadata(input, sha256sum)
	if err != nil {
		return err
	}

	err = yaml.MarshalToFile(metadata, filepath.Join(uploadPath, metadataYamlFilename))
	if err != nil {
		return err
	}

	parentDir := filepath.Dir(path)
	if err := mkdirAllNoFollow(parentDir, 0755); err != nil {
		return err
	}

	err = renameNoFollow(uploadPath, path)
	if err != nil {
		return err
	}

	return nil
}

func (b *Bucket) CopyObject(ctx context.Context, dstKey string, input core.CopyObjectInput) (*core.CopyObjectResult, error) { //nolint:funlen,lll
	dstPath, err := b.config.objectPath(b.name, dstKey)
	if err != nil {
		return nil, err
	}

	_, cancel, err := b.Locker.Lock(ctx, dstPath)
	if err != nil {
		return nil, err
	}
	defer cancel()

	if err := rejectSymlinkInPath(dstPath); err != nil {
		return nil, err
	}

	if _, err := os.Lstat(dstPath); err == nil {
		if input.IfNoneMatch {
			return nil, core.ErrPreconditionFailed
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	uploadPath := b.config.newUploadPath()

	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(uploadPath)

	srcObj := input.Source.(*Object) //nolint:forcetypeassert

	blobDst := filepath.Join(uploadPath, blobFilename)

	if err := os.Link(filepath.Join(srcObj.path, blobFilename), blobDst); err != nil {
		return nil, err
	}

	srcMeta := input.Source.Metadata()

	metadata := core.ObjectMetadata{
		SHA256:       srcMeta.SHA256,
		SHA256Base64: srcMeta.SHA256Base64,
		Size:         srcMeta.Size,
		LastModified: time.Now(),
	}

	if input.MetadataDirective == core.CopyDirectiveReplace {
		metadata.ContentType = input.ContentType
		metadata.Meta = input.ReplacementMeta
	} else {
		metadata.ContentType = srcMeta.ContentType
		metadata.Meta = srcMeta.Meta
	}

	if input.TaggingDirective == core.CopyDirectiveReplace {
		metadata.Tags = input.ReplacementTags
	} else {
		metadata.Tags = srcMeta.Tags
	}

	if err := yaml.MarshalToFile(metadata, filepath.Join(uploadPath, metadataYamlFilename)); err != nil {
		return nil, err
	}

	parentDir := filepath.Dir(dstPath)
	if err := rejectSymlinkInPath(parentDir); err != nil {
		return nil, err
	}

	if existing, _ := ObjectFromPath(b, dstKey); existing != nil {
		if err := existing.Delete(); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, err
	}

	if err := os.Rename(uploadPath, dstPath); err != nil {
		return nil, err
	}

	return &core.CopyObjectResult{Metadata: metadata}, nil
}

func (b *Bucket) GetObject(_ context.Context, key string) (core.Object, error) {
	return b.getObject(key)
}

func (b *Bucket) ListObjectsV2(ctx context.Context, input core.ListObjectsV2Input) (*core.ListV2Result, error) { //nolint:funlen,lll
	objects := []core.Object{}
	commonPrefixes := []string{}
	seenPrefixes := map[string]bool{}
	count := 0
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

	var skipPrefix string

	err := WalkBucket(ctx, b, input.Prefix, nextKey, func(_ context.Context, object core.Object) error {
		key := object.Key()

		if skipPrefix != "" && strings.HasPrefix(key, skipPrefix) {
			return nil
		}

		skipPrefix = ""

		if count >= input.MaxKeys {
			isTruncated = true
			continuationToken = lo.ToPtr(base64.StdEncoding.EncodeToString([]byte(key)))

			return StopWalk
		}

		if input.Delimiter != "" {
			rest := strings.TrimPrefix(key, input.Prefix)
			if idx := strings.Index(rest, input.Delimiter); idx >= 0 {
				cp := input.Prefix + rest[:idx+len(input.Delimiter)]
				if !seenPrefixes[cp] {
					seenPrefixes[cp] = true
					commonPrefixes = append(commonPrefixes, cp)
					count++
				}

				skipPrefix = cp

				return nil
			}
		}

		objects = append(objects, object)
		count++

		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, err
	}

	return &core.ListV2Result{
		Objects:           objects,
		CommonPrefixes:    commonPrefixes,
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

	if err := mkdirAllNoFollow(uploadPath, 0755); err != nil {
		return "", err
	}

	if err := yaml.MarshalToFile(metadata, filepath.Join(uploadPath, metadataYamlFilename)); err != nil {
		return "", err
	}

	return id, nil
}

func (b *Bucket) UploadPart(ctx context.Context, _ string, uploadID string, partNumber int, body io.Reader) (string, error) { //nolint:lll
	uploadPath, err := b.config.multipartUploadPath(uploadID)
	if err != nil {
		return "", err
	}

	path := filepath.Join(uploadPath, fmt.Sprintf("part-%d", partNumber))

	_, cancel, err := b.Locker.Lock(ctx, path)
	if err != nil {
		return "", err
	}
	defer cancel()

	if err := rejectSymlinkInPath(uploadPath); err != nil {
		return "", err
	}

	// TODO: this behavior should depend on the passed details
	if _, err := os.Lstat(path); err == nil {
		return "", core.ErrObjectAlreadyExists
	}

	uploadFile, err := createFileNoFollow(path, 0644)
	if err != nil {
		return "", err
	}
	defer uploadFile.Close()

	_, checksum, err := smartio.Copy(ctx, uploadFile, body)
	if err != nil {
		return "", err
	}

	return checksum, nil
}

func (b *Bucket) CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []core.CompletePart) (*core.ObjectMetadata, error) { //nolint:lll,funlen
	slices.SortFunc(parts, func(a, b core.CompletePart) int {
		return a.PartNumber - b.PartNumber
	})

	uploadPath, err := b.config.multipartUploadPath(uploadID)
	if err != nil {
		return nil, err
	}

	if err := rejectSymlinkInPath(uploadPath); err != nil {
		return nil, err
	}

	for _, part := range parts {
		path := filepath.Join(uploadPath, fmt.Sprintf("part-%d", part.PartNumber))
		if _, err := os.Lstat(path); err != nil {
			return nil, err
		}
	}

	blobFile, err := createFileNoFollow(filepath.Join(uploadPath, blobFilename), 0644)
	if err != nil {
		return nil, err
	}
	defer blobFile.Close()

	for _, part := range parts {
		path := filepath.Join(uploadPath, fmt.Sprintf("part-%d", part.PartNumber))

		partFile, err := openFileNoFollow(path)
		if err != nil {
			return nil, err
		}

		_, checksum, err := smartio.Copy(ctx, blobFile, partFile)
		if err != nil {
			partFile.Close()

			return nil, err
		}

		normalizedETag := strings.Trim(part.ETag, "\"")
		if normalizedETag != "" && normalizedETag != checksum {
			partFile.Close()

			return nil, fmt.Errorf("%w: part %d ETag mismatch", core.ErrObjectChecksumMismatch, part.PartNumber)
		}

		partFile.Close()
	}

	files, err := os.ReadDir(uploadPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "part-") {
			err := os.Remove(filepath.Join(uploadPath, file.Name()))
			if err != nil {
				return nil, err
			}
		}
	}

	blobFileStat, err := blobFile.Stat()
	if err != nil {
		return nil, err
	}

	metadata, err := yaml.UnmarshalFromFile[core.ObjectMetadata](filepath.Join(uploadPath, metadataYamlFilename))
	if err != nil {
		return nil, err
	}

	metadata.Size = blobFileStat.Size()

	blobPath := filepath.Join(uploadPath, blobFilename)

	blobReader, err := openFileNoFollow(blobPath)
	if err != nil {
		return nil, err
	}
	defer blobReader.Close()

	_, sha256sum, err := smartio.Copy(ctx, io.Discard, blobReader)
	if err != nil {
		return nil, err
	}

	rawSha256, err := hex.DecodeString(sha256sum)
	if err != nil {
		return nil, err
	}

	metadata.SHA256 = sha256sum
	metadata.SHA256Base64 = base64.StdEncoding.EncodeToString(rawSha256)
	metadata.LastModified = time.Now()

	err = yaml.MarshalToFile(metadata, filepath.Join(uploadPath, metadataYamlFilename))
	if err != nil {
		return nil, err
	}

	objPath, err := b.config.objectPath(b.name, key)
	if err != nil {
		return nil, err
	}

	parentDir := filepath.Dir(objPath)
	if err := rejectSymlinkInPath(parentDir); err != nil {
		return nil, err
	}

	if err := mkdirAllNoFollow(parentDir, 0755); err != nil {
		return nil, err
	}

	if err := renameNoFollow(uploadPath, objPath); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (b *Bucket) AbortMultipartUpload(_ context.Context, _ string, uploadID string) error {
	uploadPath, err := b.config.multipartUploadPath(uploadID)
	if err != nil {
		return err
	}

	if err := rejectSymlink(uploadPath); err != nil {
		return err
	}

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

func (b *Bucket) rootPath() (string, error) {
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
