package folder

import (
	"context"
	"fmt"
	"os"

	"github.com/zhulik/d3/internal/core"
)

type Backend struct {
	*BackendBuckets
	*BackendObjects

	Cfg *core.Config

	config *config
}

func (b *Backend) Init(_ context.Context) error {
	b.config = &config{b.Cfg}

	if b.config.FolderBackendPath == "" {
		return fmt.Errorf("%w: FolderBackendPath is not set", core.ErrInvalidConfig)
	}

	fileInfo, err := os.Stat(b.config.FolderBackendPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(b.config.FolderBackendPath, 0755)
		}
		return fmt.Errorf("%w: unable to access FolderBackendPath: %w", core.ErrInvalidConfig, err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("%w: FolderBackendPath is not a directory: %s", core.ErrInvalidConfig, b.config.FolderBackendPath)
	}
	if fileInfo.Mode().Perm()&(0400) == 0 {
		return fmt.Errorf("%w: FolderBackendPath is not readable: %s", core.ErrInvalidConfig, b.config.FolderBackendPath)
	}
	if fileInfo.Mode().Perm()&(0200) == 0 {
		return fmt.Errorf("%w: FolderBackendPath is not writeable: %s", core.ErrInvalidConfig, b.config.FolderBackendPath)
	}

	return nil
}
