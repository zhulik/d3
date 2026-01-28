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

	Backend           BackendType `env:"BACKEND" envDefault:"folder"`
	FolderBackendPath string      `env:"FOLDER_BACKEND_PATH" envDefault:"./d3_data"`
	RedisAddress      string      `env:"REDIS_ADDRESS" envDefault:"localhost:6379"`
}

func (c *Config) Init(_ context.Context) error {
	err := env.Parse(c)
	if err != nil {
		return fmt.Errorf("%w: failed to parse config: %w", ErrInvalidConfig, err)
	}

	return nil
}
