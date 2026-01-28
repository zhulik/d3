package folder

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/pkg/yaml"
)

var (
	defaultConfigYaml = configYaml{
		Version: 1,
	}
	ErrConfigVersionMismatch = errors.New("config version mismatch")
)

type configYaml struct {
	Version int `yaml:"version"`
}

type Backend struct {
	*BackendBuckets
	*BackendObjects

	Cfg    *core.Config
	Locker *locker.Locker

	config *config
}

func (b *Backend) Init(ctx context.Context) error {
	b.config = &config{b.Cfg}

	// Lock the backend to prevent concurrent initialization
	ctx, cancel, err := b.Locker.Lock(ctx, "folder-backend-init")
	if err != nil {
		return err
	}
	defer cancel()

	_, err = os.Stat(b.config.FolderBackendPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(b.config.FolderBackendPath, 0755)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%w: unable to access FolderBackendPath: %w", core.ErrInvalidConfig, err)
		}
	}

	return b.prepareFileStructure(ctx)
}

func (b *Backend) prepareFileStructure(ctx context.Context) error {
	err := b.prepareConfigYaml(ctx)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(b.config.bucketsPath(), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(b.config.uploadsPath(), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(b.config.binPath(), 0755); err != nil {
		return err
	}
	return nil
}

func (b *Backend) prepareConfigYaml(_ context.Context) error {
	configPath := b.config.configYamlPath()
	_, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return yaml.MarshalToFile(defaultConfigYaml, configPath)
		}
		return err
	}

	existingConfig, err := yaml.UnmarshalFromFile[configYaml](configPath)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %w", configPath, err)
	}
	if existingConfig.Version != defaultConfigYaml.Version {
		return fmt.Errorf("%w: config version mismatch: expected %d, got %d", ErrConfigVersionMismatch, defaultConfigYaml.Version, existingConfig.Version)
	}

	return nil
}
