package folder

import (
	"path/filepath"

	"github.com/zhulik/d3/internal/core"
)

const configYamlFilename = "d3.yaml"
const bucketsFolder = "buckets"
const tmpFolder = "tmp"

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

func (c *config) tmpPath() string {
	return filepath.Join(c.FolderBackendPath, tmpFolder)
}

func (c *config) configYamlPath() string {
	return filepath.Join(c.FolderBackendPath, configYamlFilename)
}
