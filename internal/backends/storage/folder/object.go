package folder

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/yaml"
)

type Object struct {
	io.ReadSeekCloser

	key string

	bucket   *Bucket
	path     string
	metadata *core.ObjectMetadata
}

func (o *Object) Key() string {
	return o.key
}

func (o *Object) LastModified() time.Time {
	return o.Metadata().LastModified
}

func (o *Object) Size() int64 {
	return o.Metadata().Size
}

func ObjectFromPath(bucket *Bucket, key string) (*Object, error) {
	path := bucket.config.objectPath(bucket.name, key)

	isObject, err := IsObjectPath(path)
	if err != nil {
		return nil, err
	}

	if !isObject {
		return nil, nil //nolint:nilnil
	}

	metadataPath := filepath.Join(path, metadataYamlFilename)

	info, err := os.Stat(metadataPath)
	if err != nil {
		return nil, err
	}

	if info.Mode().Perm()&(0400) == 0 {
		return nil, fmt.Errorf("%w: %s", core.ErrObjectMetadataNotReadable, metadataPath)
	}

	return &Object{
		bucket: bucket,
		path:   path,
		key:    key,
	}, nil
}

func (o *Object) Read(p []byte) (int, error) {
	if o.ReadSeekCloser == nil {
		rc, err := os.Open(filepath.Join(o.path, blobFilename))
		if err != nil {
			return 0, err
		}

		o.ReadSeekCloser = rc
	}

	return o.ReadSeekCloser.Read(p)
}

func (o *Object) Seek(offset int64, whence int) (int64, error) {
	if o.ReadSeekCloser == nil {
		rc, err := os.Open(filepath.Join(o.path, blobFilename))
		if err != nil {
			return 0, err
		}

		o.ReadSeekCloser = rc
	}

	return o.ReadSeekCloser.Seek(offset, whence)
}

func (o *Object) Close() error {
	if o.ReadSeekCloser == nil {
		return nil
	}

	return o.ReadSeekCloser.Close()
}

func (o *Object) Metadata() *core.ObjectMetadata {
	if o.metadata == nil {
		metadata := lo.Must(yaml.UnmarshalFromFile[core.ObjectMetadata](filepath.Join(o.path, metadataYamlFilename)))
		o.metadata = &metadata
	}

	return o.metadata
}

func (o *Object) Delete() error {
	err := os.Rename(o.path, o.bucket.config.newBinPath())
	if err != nil {
		return err
	}

	root := filepath.Clean(o.bucket.rootPath())
	for parent := filepath.Clean(filepath.Dir(o.path)); parent != root; parent = filepath.Clean(filepath.Dir(parent)) {
		entries, err := os.ReadDir(parent)
		if err != nil || len(entries) != 0 {
			return err
		}
		// we ignore the error on purpose because there might be concurrent uploads to the same directory
		os.Remove(parent)
	}

	return nil
}

func IsObjectPath(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, core.ErrObjectNotFound
		}

		return false, err
	}

	if !fi.IsDir() {
		return false, nil
	}

	exists, err := existsAndIsFile(filepath.Join(path, blobFilename))
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	exists, err = existsAndIsFile(filepath.Join(path, metadataYamlFilename))
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	return true, nil
}

func existsAndIsFile(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return !fi.IsDir(), nil
}
