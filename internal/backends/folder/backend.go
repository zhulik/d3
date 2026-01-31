package folder

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/pkg/credentials"
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
	*BackendBuckets
	*BackendObjects

	Locker *locker.Locker

	Config *Config
}

func (b *Backend) Init(ctx context.Context) error {
	// Lock the backend to prevent concurrent initialization
	ctx, cancel, err := b.Locker.Lock(ctx, "folder-backend-init")
	if err != nil {
		return err
	}
	defer cancel()

	_, err = os.Stat(b.Config.FolderBackendPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(b.Config.FolderBackendPath, 0755)
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

	if err := os.MkdirAll(b.Config.bucketsPath(), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(b.Config.uploadsPath(), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(b.Config.binPath(), 0755); err != nil {
		return err
	}
	return nil
}

func (b *Backend) prepareConfigYaml(_ context.Context) error {
	configPath := b.Config.configYamlPath()
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
		return fmt.Errorf("%w: config version mismatch: expected %d, got %d", ErrConfigVersionMismatch, ConfigVersion, existingConfig.Version)
	}

	return nil
}
