package folder

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/yaml"
)

type Object struct {
	io.ReadSeekCloser

	config       *Config
	bucket       string
	path         string
	blobPath     string
	metadataPath string
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

	return &Object{
		config:       cfg,
		path:         path,
		bucket:       bucket,
		blobPath:     filepath.Join(path, blobFilename),
		metadataPath: filepath.Join(path, metadataYamlFilename),
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

func (o *Object) Metadata() (*core.ObjectMetadata, error) {
	metadata, err := yaml.UnmarshalFromFile[core.ObjectMetadata](o.metadataPath)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
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
			return false, common.ErrObjectNotFound
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
