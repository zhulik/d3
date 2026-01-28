package folder

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"go.yaml.in/yaml/v3"
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

	ctx, cancel, err := b.Locker.Lock(ctx, "folder-backend-init")
	if err != nil {
		return err
	}
	defer cancel()

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

	return b.prepareFileStructure(ctx)
}

func (b *Backend) prepareFileStructure(ctx context.Context) error {
	err := b.prepareVersionFile(ctx)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(b.config.bucketsPath(), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(b.config.tmpPath(), 0755); err != nil {
		return err
	}
	return nil
}

func (b *Backend) prepareVersionFile(_ context.Context) error {
	configPath := b.config.configYamlPath()
	_, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			yamlData, err := yaml.Marshal(defaultConfigYaml)
			if err != nil {
				return err
			}
			return os.WriteFile(configPath, yamlData, 0600)
		}
		return err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	var existingConfig configYaml
	err = yaml.Unmarshal(data, &existingConfig)
	if err != nil {
		return err
	}
	if existingConfig.Version != defaultConfigYaml.Version {
		return fmt.Errorf("%w: config version mismatch: expected %d, got %d", ErrConfigVersionMismatch, defaultConfigYaml.Version, existingConfig.Version)
	}

	return nil
}
