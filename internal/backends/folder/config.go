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

func (c *config) binPath() string {
	return filepath.Join(c.FolderBackendPath, tmpFolder, binFolder)
}

func (c *config) newBinPath() string {
	return filepath.Join(c.binPath(), uuid.NewString())
}

func (c *config) configYamlPath() string {
	return filepath.Join(c.FolderBackendPath, configYamlFilename)
}

func (c *config) newUploadPath() string {
	return filepath.Join(c.uploadsPath(), uuid.NewString())
}

func (c *config) multipartUploadPath(uploadID string) string {
	return filepath.Join(c.uploadsPath(), fmt.Sprintf("multipart-%s", uploadID))
}

func (c *config) newMultipartUploadPath() (string, string) {
	id := uuid.NewString()
	return id, filepath.Join(c.uploadsPath(), fmt.Sprintf("multipart-%s", id))
}
