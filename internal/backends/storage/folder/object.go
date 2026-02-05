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

	key          string
	lastModified time.Time
	size         int64

	config   *Config
	bucket   string
	path     string
	blobPath string
	metadata *core.ObjectMetadata
}

func (o *Object) Key() string {
	return o.key
}

func (o *Object) LastModified() time.Time {
	return o.lastModified
}

func (o *Object) Size() int64 {
	return o.size
}

func ObjectFromPath(cfg *Config, bucket, key string) (*Object, error) {
	path := cfg.objectPath(bucket, key)

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
		config:   cfg,
		path:     path,
		bucket:   bucket,
		blobPath: filepath.Join(path, blobFilename),
	}, nil
}

func (o *Object) Read(p []byte) (int, error) {
	if o.ReadSeekCloser == nil {
		rc, err := os.Open(o.blobPath)
		if err != nil {
			return 0, err
		}

		o.ReadSeekCloser = rc
	}

	return o.ReadSeekCloser.Read(p)
}

func (o *Object) Seek(offset int64, whence int) (int64, error) {
	if o.ReadSeekCloser == nil {
		rc, err := os.Open(o.blobPath)
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
		ometadata := lo.Must(yaml.UnmarshalFromFile[core.ObjectMetadata](filepath.Join(o.path, metadataYamlFilename)))
		o.metadata = &ometadata
	}

	return o.metadata
}

func (o *Object) Delete() error {
	err := os.Rename(o.path, o.config.newBinPath())
	if err != nil {
		return err
	}

	parentDir := filepath.Dir(o.path)

	entries, readErr := os.ReadDir(parentDir)
	if readErr == nil && len(entries) == 0 {
		// we ignore the on purpose because there might be concurrent uploads to the same directory
		if parentDir != o.config.bucketPath(o.bucket) {
			os.Remove(parentDir)
		}
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
