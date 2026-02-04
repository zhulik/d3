package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/caarlos0/env/v11"
)

var (
	ErrInvalidConfig = errors.New("invalid config")
)

type StorageBackendType string

const (
	StorageBackendFolder StorageBackendType = "folder"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"production"`

	StorageBackend           StorageBackendType `env:"STORAGE_BACKEND"             envDefault:"folder"`
	FolderStorageBackendPath string             `env:"FOLDER_STORAGE_BACKEND_PATH" envDefault:"./d3_data"`

	RedisAddress string `env:"REDIS_ADDRESS" envDefault:"localhost:6379"`

	Port            int `env:"PORT"              envDefault:"8080"`
	HealthCheckPort int `env:"HEALTH_CHECK_PORT" envDefault:"8081"`
	ManagementPort  int `env:"MANAGEMENT_PORT"   envDefault:"8082"`
}

func (c *Config) Init(_ context.Context) error {
	if c.Port != 0 {
		// already initialized manually
		return nil
	}

	err := env.Parse(c)
	if err != nil {
		return fmt.Errorf("%w: failed to parse config: %w", ErrInvalidConfig, err)
	}

	if c.StorageBackend == StorageBackendFolder {
		if c.FolderStorageBackendPath == "" {
			return fmt.Errorf("%w: FolderStorageBackendPath is not set", ErrInvalidConfig)
		}
	} else {
		return fmt.Errorf("%w: unknown backend: %s", ErrInvalidConfig, c.StorageBackend)
	}

	return nil
}
