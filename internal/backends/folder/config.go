package folder

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/zhulik/d3/internal/core"
)

const (
	configYamlFilename   = "d3.yaml"
	bucketsFolder        = "buckets"
	tmpFolder            = "tmp"
	uploadsFolder        = "uploads"
	metadataYamlFilename = "metadata.yaml"
	blobFilename         = "blob"
	binFolder            = "bin"
)

type Config struct {
	*core.Config
}

func (c *Config) bucketPath(bucket string) string {
	return filepath.Join(c.FolderBackendPath, bucketsFolder, bucket)
}

func (c *Config) objectPath(bucket, key string) string {
	return filepath.Join(c.FolderBackendPath, bucketsFolder, bucket, key)
}

func (c *Config) bucketsPath() string {
	return filepath.Join(c.FolderBackendPath, bucketsFolder)
}

func (c *Config) uploadsPath() string {
	return filepath.Join(c.FolderBackendPath, tmpFolder, uploadsFolder)
}

func (c *Config) binPath() string {
	return filepath.Join(c.FolderBackendPath, tmpFolder, binFolder)
}

func (c *Config) newBinPath() string {
	return filepath.Join(c.binPath(), uuid.NewString())
}

func (c *Config) configYamlPath() string {
	return filepath.Join(c.FolderBackendPath, configYamlFilename)
}

func (c *Config) tmpPath() string {
	return filepath.Join(c.FolderBackendPath, tmpFolder)
}

func (c *Config) newUploadPath() string {
	return filepath.Join(c.uploadsPath(), uuid.NewString())
}

func (c *Config) multipartUploadPath(uploadID string) string {
	return filepath.Join(c.uploadsPath(), fmt.Sprintf("multipart-%s", uploadID))
}

func (c *Config) newMultipartUploadPath() (string, string) {
	id := uuid.NewString()
	return id, filepath.Join(c.uploadsPath(), fmt.Sprintf("multipart-%s", id))
}
