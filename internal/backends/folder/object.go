package folder

import (
	"io"
	"os"
	"path/filepath"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/yaml"
)

type Object struct {
	Path string
}

func ObjectFromPath(path string) (*Object, error) {
	isObject, err := IsObjectPath(path)
	if err != nil {
		return nil, err
	}
	if !isObject {
		return nil, nil
	}

	return &Object{
		Path: path,
	}, nil
}

func (o *Object) Open() (io.ReadCloser, error) {
	return os.Open(filepath.Join(o.Path, blobFilename))
}

func (o *Object) Metadata() (*core.ObjectMetadata, error) {
	metadata, err := yaml.UnmarshalFromFile[core.ObjectMetadata](filepath.Join(o.Path, metadataYamlFilename))
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (o *Object) Delete() error {
	return os.RemoveAll(o.Path)
}

func IsObjectPath(path string) (bool, error) {
	fi, err := os.Stat(filepath.Join(path))
	if err != nil {
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
		return false, err
	}
	return !fi.IsDir(), nil
}
