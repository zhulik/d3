package yaml

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/backends/storage/folder"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/atomicwriter"
	"github.com/zhulik/d3/pkg/credentials"
	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/yaml"
)

const (
	pollInterval = 5 * time.Second
)

type Backend struct {
	Config *core.Config
	Locker core.Locker
	Logger *slog.Logger

	lastUpdated        time.Time
	adminUser          *core.User
	usersByName        map[string]*core.User
	usersByAccessKeyID map[string]*core.User
	policiesByID       map[string]*iampol.IAMPolicy

	rwLock sync.RWMutex
	writer *atomicwriter.AtomicWriter
}

func (b *Backend) Init(ctx context.Context) error {
	// Lock the backend to prevent concurrent initialization
	ctx, cancel, err := b.Locker.Lock(ctx, "yaml-management-backend-init")
	if err != nil {
		return err
	}
	defer cancel()

	err = os.MkdirAll(filepath.Dir(b.Config.ManagementBackendTmpPath), 0755)
	if err != nil {
		return err
	}

	b.writer = atomicwriter.New(b.Locker, filepath.Join(b.Config.ManagementBackendTmpPath, folder.TmpFolder))

	managementConfigPath := b.Config.ManagementBackendYAMLPath

	err = os.MkdirAll(filepath.Dir(managementConfigPath), 0755)
	if err != nil {
		return err
	}

	_, err = os.Stat(managementConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			accessKeyID, secretAccessKey := credentials.GenerateCredentials()
			cfg := ManagementConfig{
				Version: ConfigVersion,
				AdminUser: user{
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccessKey,
				},
				Policies: map[string]*iampol.IAMPolicy{},
			}

			err := yaml.MarshalToFile(cfg, managementConfigPath)
			if err != nil {
				return err
			}

			return b.reload(ctx)
		}

		return err
	}

	existingConfig, err := yaml.UnmarshalFromFile[ManagementConfig](managementConfigPath)
	if err != nil {
		return fmt.Errorf("failed to unmarshal management backend config file %s: %w", managementConfigPath, err)
	}

	if existingConfig.Version != ConfigVersion {
		return fmt.Errorf("%w: management config version mismatch: expected %d, got %d",
			core.ErrConfigVersionMismatch, ConfigVersion, existingConfig.Version)
	}

	return b.reload(ctx)
}

func (b *Backend) GetUsers(_ context.Context) ([]string, error) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	return append(lo.Keys(b.usersByName), "admin"), nil
}

func (b *Backend) GetUserByName(_ context.Context, name string) (*core.User, error) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	if name == b.adminUser.Name {
		return b.adminUser, nil
	}

	user, ok := b.usersByName[name]
	if !ok {
		return nil, core.ErrUserNotFound
	}

	return user, nil
}

func (b *Backend) GetUserByAccessKeyID(_ context.Context, accessKeyID string) (*core.User, error) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	if accessKeyID == b.adminUser.AccessKeyID {
		return b.adminUser, nil
	}

	user, ok := b.usersByAccessKeyID[accessKeyID]
	if !ok {
		return nil, core.ErrUserNotFound
	}

	return user, nil
}

func (b *Backend) CreateUser(ctx context.Context, newUser *core.User) error {
	if newUser.Name == "" || newUser.AccessKeyID == "" || newUser.SecretAccessKey == "" {
		return core.ErrUserInvalid
	}

	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		if _, ok := cfg.Users[newUser.Name]; ok {
			return cfg, core.ErrUserAlreadyExists
		}

		cfg.Users[newUser.Name] = &user{
			AccessKeyID:     newUser.AccessKeyID,
			SecretAccessKey: newUser.SecretAccessKey,
		}

		return cfg, nil
	})
}

func (b *Backend) UpdateUser(ctx context.Context, updatedUser *core.User) error {
	if updatedUser.Name == "" || updatedUser.AccessKeyID == "" || updatedUser.SecretAccessKey == "" {
		return core.ErrUserInvalid
	}

	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		if _, ok := cfg.Users[updatedUser.Name]; !ok {
			return cfg, core.ErrUserNotFound
		}

		cfg.Users[updatedUser.Name] = &user{
			AccessKeyID:     updatedUser.AccessKeyID,
			SecretAccessKey: updatedUser.SecretAccessKey,
		}

		return cfg, nil
	})
}

func (b *Backend) DeleteUser(ctx context.Context, userName string) error {
	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		if _, ok := cfg.Users[userName]; !ok {
			return cfg, core.ErrUserNotFound
		}

		delete(cfg.Users, userName)

		return cfg, nil
	})
}

func (b *Backend) AdminCredentials() (string, string) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	return b.adminUser.AccessKeyID, b.adminUser.SecretAccessKey
}

func (b *Backend) GetPolicies(_ context.Context) ([]string, error) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	return lo.Keys(b.policiesByID), nil
}

func (b *Backend) GetPolicyByID(_ context.Context, id string) (*iampol.IAMPolicy, error) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	policy, ok := b.policiesByID[id]
	if !ok {
		return nil, core.ErrPolicyNotFound
	}

	return policy, nil
}

func (b *Backend) CreatePolicy(ctx context.Context, newPolicy *iampol.IAMPolicy) error {
	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		if _, ok := cfg.Policies[newPolicy.ID]; ok {
			return cfg, core.ErrPolicyAlreadyExists
		}

		cfg.Policies[newPolicy.ID] = newPolicy

		return cfg, nil
	})
}

func (b *Backend) UpdatePolicy(ctx context.Context, updatedPolicy *iampol.IAMPolicy) error {
	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		if _, ok := cfg.Policies[updatedPolicy.ID]; !ok {
			return cfg, core.ErrPolicyNotFound
		}

		cfg.Policies[updatedPolicy.ID] = updatedPolicy

		return cfg, nil
	})
}

func (b *Backend) DeletePolicy(ctx context.Context, policyID string) error {
	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		if _, ok := cfg.Policies[policyID]; !ok {
			return cfg, core.ErrPolicyNotFound
		}

		delete(cfg.Policies, policyID)

		return cfg, nil
	})
}

// Run watches the main config file and reloads the user repository when it changes.
func (b *Backend) Run(ctx context.Context) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	errorsCount := 0

	var allErrors error

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := b.checkAndReload(ctx)
			if err != nil {
				errorsCount++
				allErrors = errors.Join(allErrors, err)
				b.Logger.Error("failed to check and reload user repository", "error", err)
			}

			if errorsCount > 3 {
				return fmt.Errorf("failed to check and reload user repository after 3 attempts: %w", allErrors)
			}
		}
	}
}

func (b *Backend) readWriteConfig(ctx context.Context, op func(ManagementConfig) (ManagementConfig, error)) error {
	err := b.writer.ReadWrite(ctx, b.Config.ManagementBackendYAMLPath,
		func(_ context.Context, content []byte) ([]byte, error) {
			managementConfig, err := yaml.Unmarshal[ManagementConfig](content)
			if err != nil {
				return nil, err
			}

			modifiedConfig, err := op(managementConfig)
			if err != nil {
				return nil, err
			}

			return yaml.Marshal(modifiedConfig)
		})
	if err != nil {
		return err
	}

	return b.reload(ctx)
}

func (b *Backend) checkAndReload(ctx context.Context) error {
	path := b.Config.ManagementBackendYAMLPath

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	b.rwLock.RLock()

	if b.lastUpdated.IsZero() || info.ModTime() != b.lastUpdated {
		b.rwLock.RUnlock()

		b.Logger.Info("config file changed, reloading user repository")

		return b.reload(ctx)
	}

	b.rwLock.RUnlock()

	return nil
}

func (b *Backend) reload(ctx context.Context) error {
	b.rwLock.Lock()
	defer b.rwLock.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	path := b.Config.ManagementBackendYAMLPath

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	managementConfig, err := yaml.UnmarshalFromFile[ManagementConfig](path)
	if err != nil {
		return err
	}

	b.usersByName = map[string]*core.User{}
	b.usersByAccessKeyID = map[string]*core.User{}
	b.policiesByID = map[string]*iampol.IAMPolicy{}
	b.lastUpdated = info.ModTime()
	b.adminUser = managementConfig.AdminUser.toCoreUser("admin")

	for userName, user := range managementConfig.Users {
		b.usersByName[userName] = user.toCoreUser(userName)
		b.usersByAccessKeyID[user.AccessKeyID] = b.usersByName[userName]
	}

	maps.Copy(b.policiesByID, managementConfig.Policies)

	return nil
}
