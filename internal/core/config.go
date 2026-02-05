package core

import (
	"context"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type StorageBackendType string

const (
	StorageBackendFolder StorageBackendType = "folder"
)

type ManagementBackendType string

const (
	ManagementBackendYAML ManagementBackendType = "YAML"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"production"`

	StorageBackend           StorageBackendType `env:"STORAGE_BACKEND"             envDefault:"folder"`
	FolderStorageBackendPath string             `env:"FOLDER_STORAGE_BACKEND_PATH" envDefault:"./d3_data"`

	ManagementBackend         ManagementBackendType `env:"MANAGEMENT_BACKEND"           envDefault:"YAML"`
	ManagementBackendYAMLPath string                `env:"MANAGEMENT_BACKEND_YAML_PATH" envDefault:"./d3_data/management.yaml"` //nolint:lll
	// ManagementBackendTmpPath specifies where to store temporary files for management backend operations.
	// It should be on the same disk as the main storage to ensure atomicity. Only relevand for the YAML backend.
	ManagementBackendTmpPath string `env:"MANAGEMENT_BACKEND_TMP_PATH" envDefault:"./d3_data/tmp"`

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
