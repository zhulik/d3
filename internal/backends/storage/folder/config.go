package folder

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/zhulik/d3/internal/core"
)

const (
	configYamlFilename   = "d3.yaml"
	bucketsFolder        = "buckets"
	objectsFolder        = "objects"
	TmpFolder            = "tmp"
	uploadsFolder        = "uploads"
	regularUploadsFolder = "regular"
	multipartFolder      = "multipart"
	metadataYamlFilename = "metadata.yaml"
	blobFilename         = "blob"
	binFolder            = "bin"
)

type Config struct {
	*core.Config
}

func (c *Config) bucketPath(bucket string) (string, error) {
	path := filepath.Join(c.FolderStorageBackendPath, bucketsFolder, bucket)

	return path, EnsureContained(path, c.bucketsPath())
}

func (c *Config) objectPath(bucket, key string) (string, error) {
	bucketRoot, err := c.bucketPath(bucket)
	if err != nil {
		return "", err
	}

	objectsRoot := filepath.Join(bucketRoot, objectsFolder)

	path := filepath.Join(objectsRoot, key)

	return path, EnsureContained(path, objectsRoot)
}

func (c *Config) bucketsPath() string {
	return filepath.Join(c.FolderStorageBackendPath, bucketsFolder)
}

func (c *Config) bucketUploadsPath(bucket string) (string, error) {
	bucketRoot, err := c.bucketPath(bucket)
	if err != nil {
		return "", err
	}

	uploadsRoot := filepath.Join(bucketRoot, uploadsFolder)

	return uploadsRoot, EnsureContained(uploadsRoot, bucketRoot)
}

func (c *Config) multipartUploadsRoot(bucket string) (string, error) {
	uploadsRoot, err := c.bucketUploadsPath(bucket)
	if err != nil {
		return "", err
	}

	root := filepath.Join(uploadsRoot, multipartFolder)

	return root, EnsureContained(root, uploadsRoot)
}

func (c *Config) binPath() string {
	return filepath.Join(c.FolderStorageBackendPath, TmpFolder, binFolder)
}

func (c *Config) newBinPath() string {
	return filepath.Join(c.binPath(), uuid.NewString())
}

func (c *Config) configYamlPath() string {
	return filepath.Join(c.FolderStorageBackendPath, configYamlFilename)
}

func (c *Config) newUploadPath(bucket string) (string, error) {
	uploadsRoot, err := c.bucketUploadsPath(bucket)
	if err != nil {
		return "", err
	}

	return filepath.Join(uploadsRoot, regularUploadsFolder, uuid.NewString()), nil
}

func (c *Config) newMultipartUploadPath(bucket, key string) (string, string, error) {
	root, err := c.multipartUploadsRoot(bucket)
	if err != nil {
		return "", "", err
	}

	id := uuid.NewString()

	path := filepath.Join(root, key, id)

	return id, path, EnsureContained(path, root)
}

func EnsureContained(path, parent string) error {
	cleanPath := filepath.Clean(path) + string(filepath.Separator)
	cleanParent := filepath.Clean(parent) + string(filepath.Separator)

	if !strings.HasPrefix(cleanPath, cleanParent) {
		return fmt.Errorf("%w: %q escapes %q", core.ErrPathTraversal, path, parent)
	}

	return nil
}
