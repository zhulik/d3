package core

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
)

var (
	ErrInvalidConfig = errors.New("invalid config")
)

type BackendType string

const (
	BackendFolder BackendType = "folder"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"production"`

	Backend           BackendType `env:"BACKEND" envDefault:"folder"`
	FolderBackendPath string      `env:"FOLDER_BACKEND_PATH" envDefault:"./d3_data"`
	RedisAddress      string      `env:"REDIS_ADDRESS" envDefault:"localhost:6379"`
}

func (c *Config) Init(ctx context.Context) error {
	err := env.Parse(c)
	if err != nil {
		return fmt.Errorf("%w: failed to parse config: %w", ErrInvalidConfig, err)
	}

	switch c.Backend {
	case BackendFolder:
		return c.validateFolderBackendPath()
	default:
		return fmt.Errorf("%w: unknown backend: %s", ErrInvalidConfig, c.Backend)
	}
}

func (c *Config) validateFolderBackendPath() error {
	if c.FolderBackendPath == "" {
		return fmt.Errorf("%w: FolderBackendPath is not set", ErrInvalidConfig)
	}

	fileInfo, err := os.Stat(c.FolderBackendPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(c.FolderBackendPath, 0755)
		} else {
			return fmt.Errorf("%w: unable to access FolderBackendPath: %w", ErrInvalidConfig, err)
		}
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("%w: FolderBackendPath is not a directory: %s", ErrInvalidConfig, c.FolderBackendPath)
	}
	if fileInfo.Mode().Perm()&(0400) == 0 {
		return fmt.Errorf("%w: FolderBackendPath is not readable: %s", ErrInvalidConfig, c.FolderBackendPath)
	}
	if fileInfo.Mode().Perm()&(0200) == 0 {
		return fmt.Errorf("%w: FolderBackendPath is not writeable: %s", ErrInvalidConfig, c.FolderBackendPath)
	}

	return nil
}
