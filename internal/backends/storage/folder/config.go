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
	TmpFolder            = "tmp"
	uploadsFolder        = "uploads"
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

	path := filepath.Join(c.FolderStorageBackendPath, bucketsFolder, bucket, key)

	return path, EnsureContained(path, bucketRoot)
}

func (c *Config) bucketsPath() string {
	return filepath.Join(c.FolderStorageBackendPath, bucketsFolder)
}

func (c *Config) uploadsPath() string {
	return filepath.Join(c.FolderStorageBackendPath, TmpFolder, uploadsFolder)
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

func (c *Config) newUploadPath() string {
	return filepath.Join(c.uploadsPath(), uuid.NewString())
}

func (c *Config) multipartUploadPath(uploadID string) (string, error) {
	path := filepath.Join(c.uploadsPath(), "multipart-"+uploadID)

	return path, EnsureContained(path, c.uploadsPath())
}

func (c *Config) newMultipartUploadPath() (string, string) {
	id := uuid.NewString()

	return id, filepath.Join(c.uploadsPath(), "multipart-"+id)
}

func EnsureContained(path, parent string) error {
	cleanPath := filepath.Clean(path) + string(filepath.Separator)
	cleanParent := filepath.Clean(parent) + string(filepath.Separator)

	if !strings.HasPrefix(cleanPath, cleanParent) {
		return fmt.Errorf("%w: %q escapes %q", core.ErrPathTraversal, path, parent)
	}

	return nil
}
