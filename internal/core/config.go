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

type BackendType string

const (
	BackendFolder BackendType = "folder"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"production"`

	Backend           BackendType `env:"BACKEND"             envDefault:"folder"`
	FolderBackendPath string      `env:"FOLDER_BACKEND_PATH" envDefault:"./d3_data"`
	RedisAddress      string      `env:"REDIS_ADDRESS"       envDefault:"localhost:6379"`
	Port              int         `env:"PORT"                envDefault:"8080"`
	HealthCheckPort   int         `env:"HEALTH_CHECK_PORT"   envDefault:"8081"`
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

	if c.Backend == BackendFolder {
		if c.FolderBackendPath == "" {
			return fmt.Errorf("%w: FolderBackendPath is not set", ErrInvalidConfig)
		}
	} else {
		return fmt.Errorf("%w: unknown backend: %s", ErrInvalidConfig, c.Backend)
	}

	return nil
}
