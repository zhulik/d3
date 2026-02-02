package folder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/pkg/credentials"
	"github.com/zhulik/d3/pkg/iter"
	"github.com/zhulik/d3/pkg/yaml"
)

var (
	ErrConfigVersionMismatch = errors.New("config version mismatch")
)

const (
	ConfigVersion = 1
)

type user struct {
	Name            string `yaml:"name"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

func (u user) toCoreUser() core.User {
	return core.User{
		Name:            u.Name,
		AccessKeyID:     u.AccessKeyID,
		SecretAccessKey: u.SecretAccessKey,
	}
}

type configYaml struct {
	Version   int    `yaml:"version"`
	AdminUser user   `yaml:"admin_user"`
	Users     []user `yaml:"users"`
}

type Backend struct {
	Cfg *core.Config

	Locker *locker.Locker

	config *Config
}

func (b *Backend) Init(ctx context.Context) error {
	b.config = &Config{b.Cfg}

	// Lock the backend to prevent concurrent initialization
	ctx, cancel, err := b.Locker.Lock(ctx, "folder-backend-init")
	if err != nil {
		return err
	}
	defer cancel()

	_, err = os.Stat(b.Cfg.FolderBackendPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(b.Cfg.FolderBackendPath, 0755)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%w: unable to access FolderBackendPath: %w", core.ErrInvalidConfig, err)
		}
	}

	return b.prepareFileStructure(ctx)
}

func (b *Backend) ListBuckets(_ context.Context) ([]core.Bucket, error) {
	entries, err := os.ReadDir(b.config.bucketsPath())
	if err != nil {
		return nil, err
	}

	return iter.ErrMap(entries, b.dirEntryToBucket)
}

func (b *Backend) CreateBucket(_ context.Context, name string) error {
	path := b.config.bucketPath(name)

	err := os.Mkdir(path, 0755)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return common.ErrBucketAlreadyExists
		}

		return err
	}

	return nil
}

func (b *Backend) DeleteBucket(_ context.Context, name string) error {
	path := b.config.bucketPath(name)

	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return common.ErrBucketNotFound
		}

		var pathError *os.PathError
		if errors.As(err, &pathError) {
			if errors.Is(pathError.Err, syscall.ENOTEMPTY) {
				return common.ErrBucketNotEmpty
			}
		}

		return err
	}

	return nil
}

func (b *Backend) HeadBucket(_ context.Context, name string) (core.Bucket, error) {
	path := b.config.bucketPath(name)

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, common.ErrBucketNotFound
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

func (b *Backend) dirEntryToBucket(entry os.DirEntry) (core.Bucket, error) {
	info, err := entry.Info()
	if err != nil {
		return nil, err
	}

	return &Bucket{
		name:         entry.Name(),
		creationDate: info.ModTime(), // TODO: use the actual creation date
		config:       b.config,
		Locker:       b.Locker,
	}, nil
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
			accessKeyID, secretAccessKey := credentials.GenerateCredentials()
			cfg := configYaml{
				Version: ConfigVersion,
				AdminUser: user{
					Name:            "admin",
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccessKey,
				},
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
		return fmt.Errorf("%w: config version mismatch: expected %d, got %d",
			ErrConfigVersionMismatch, ConfigVersion, existingConfig.Version)
	}

	return nil
}
