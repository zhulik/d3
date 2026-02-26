package folder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/xiter"
	"github.com/zhulik/d3/pkg/yaml"
)

const (
	ConfigVersion = 1
)

type configYaml struct {
	Version int `yaml:"version"`
}

type Backend struct {
	Cfg *core.Config

	Locker core.Locker

	config *Config
}

func (b *Backend) Init(ctx context.Context) error {
	b.config = &Config{b.Cfg}

	// Lock the backend to prevent concurrent initialization
	ctx, cancel, err := b.Locker.Lock(ctx, "folder-storage-backend-init")
	if err != nil {
		return err
	}
	defer cancel()

	_, err = os.Stat(b.Cfg.FolderStorageBackendPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(b.Cfg.FolderStorageBackendPath, 0755)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%w: unable to access FolderStorageBackendPath: %w", core.ErrInvalidConfig, err)
		}
	}

	return b.prepareFileStructure(ctx)
}

func (b *Backend) ListBuckets(_ context.Context) ([]core.Bucket, error) {
	entries, err := os.ReadDir(b.config.bucketsPath())
	if err != nil {
		return nil, err
	}

	return xiter.ErrFilterMap(entries, b.dirEntryToBucket)
}

func (b *Backend) CreateBucket(_ context.Context, name string) error {
	path, err := b.config.bucketPath(name)
	if err != nil {
		return err
	}

	err = os.Mkdir(path, 0755)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return core.ErrBucketAlreadyExists
		}

		return err
	}

	return nil
}

func (b *Backend) DeleteBucket(_ context.Context, name string) error {
	path, err := b.config.bucketPath(name)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return core.ErrBucketNotFound
		}

		var pathError *os.PathError
		if errors.As(err, &pathError) {
			if errors.Is(pathError.Err, syscall.ENOTEMPTY) {
				return core.ErrBucketNotEmpty
			}
		}

		return err
	}

	return nil
}

func (b *Backend) HeadBucket(_ context.Context, name string) (core.Bucket, error) {
	path, err := b.config.bucketPath(name)
	if err != nil {
		return nil, err
	}

	if err := rejectSymlink(path); err != nil {
		return nil, err
	}

	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, core.ErrBucketNotFound
		}

		return nil, err
	}

	return &Bucket{
		name:         name,
		creationDate: info.ModTime(), // TODO: use the actual creation date
		config:       b.config,
		Locker:       b.Locker,
	}, nil
}

func (b *Backend) dirEntryToBucket(entry os.DirEntry) (core.Bucket, bool, error) {
	if entry.Type()&os.ModeSymlink != 0 {
		return nil, false, nil
	}

	info, err := entry.Info()
	if err != nil {
		return nil, false, err
	}

	return &Bucket{
		name:         entry.Name(),
		creationDate: info.ModTime(), // TODO: use the actual creation date
		config:       b.config,
		Locker:       b.Locker,
	}, true, nil
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
			cfg := configYaml{
				Version: ConfigVersion,
			}

			err := yaml.MarshalToFile(cfg, configPath)
			if err != nil {
				return err
			}

			return nil
		}

		return err
	}

	existingConfig, err := yaml.UnmarshalFromFile[configYaml](configPath)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %w", configPath, err)
	}

	if existingConfig.Version != ConfigVersion {
		return fmt.Errorf("%w: backend config version mismatch: expected %d, got %d",
			core.ErrConfigVersionMismatch, ConfigVersion, existingConfig.Version)
	}

	return nil
}
