package folder

import (
	"path/filepath"

	"github.com/zhulik/d3/internal/core"
)

const configYamlPath = "d3.yaml"
const bucketsPath = "buckets"
const metadataPath = "metadata"
const tmpPath = "tmp"

type config struct {
	*core.Config
}

func (c *config) bucketPath(bucket string) string {
	return filepath.Join(c.FolderBackendPath, bucketsPath, bucket)
}

func (c *config) objectPath(bucket, key string) string {
	return filepath.Join(c.FolderBackendPath, bucketsPath, bucket, key)
}

func (c *config) bucketsPath() string {
	return filepath.Join(c.FolderBackendPath, bucketsPath)
}

func (c *config) metadataPath() string {
	return filepath.Join(c.FolderBackendPath, metadataPath)
}

func (c *config) tmpPath() string {
	return filepath.Join(c.FolderBackendPath, tmpPath)
}

func (c *config) configYamlPath() string {
	return filepath.Join(c.FolderBackendPath, configYamlPath)
}
