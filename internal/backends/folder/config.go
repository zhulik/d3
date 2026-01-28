package folder

import (
	"path/filepath"

	"github.com/zhulik/d3/internal/core"
)

type config struct {
	*core.Config
}

func (c *config) bucketPath(bucket string) string {
	return filepath.Join(c.FolderBackendPath, bucket)
}

func (c *config) objectPath(bucket, key string) string {
	return filepath.Join(c.bucketPath(bucket), key)
}
