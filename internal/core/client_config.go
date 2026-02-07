package core

import (
	"context"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type ClientConfig struct {
	ServerURL       string `env:"D3_SERVER_URL"         envDefault:"http://localhost:8082"`
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"     envRequired:"true"`
	AccessKeySecret string `env:"AWS_ACCESS_KEY_SECRET" envRequired:"true"`
}

func (c *ClientConfig) Init(_ context.Context) error {
	if c.ServerURL != "" {
		// already initialized manually
		return nil
	}

	err := env.Parse(c)
	if err != nil {
		return fmt.Errorf("%w: failed to parse config: %w", ErrInvalidConfig, err)
	}

	return nil
}
