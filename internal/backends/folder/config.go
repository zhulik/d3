package folder

import (
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
)

type config struct {
	*core.Config
}

func (c *config) bucketPath(bucket string) string {
	return filepath.Join(c.FolderBackendPath, bucketsFolder, bucket)
}

func (c *config) objectPath(bucket, key string) string {
	return filepath.Join(c.FolderBackendPath, bucketsFolder, bucket, key)
}

func (c *config) bucketsPath() string {
	return filepath.Join(c.FolderBackendPath, bucketsFolder)
}

func (c *config) uploadsPath() string {
	return filepath.Join(c.FolderBackendPath, tmpFolder, uploadsFolder)
}

func (c *config) configYamlPath() string {
	return filepath.Join(c.FolderBackendPath, configYamlFilename)
}

func (c *config) newUploadPath() string {
	return filepath.Join(c.FolderBackendPath, tmpFolder, uploadsFolder, uuid.New().String())
}
