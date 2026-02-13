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

	policiesByID map[string]*iampol.IAMPolicy

	bindings         []*core.PolicyBinding
	bindingsByUser   map[string][]*core.PolicyBinding
	bindingsByPolicy map[string][]*core.PolicyBinding

	rwLock sync.RWMutex
	writer *atomicwriter.AtomicWriter
}

func (b *Backend) Init(ctx context.Context) error { //nolint:funlen
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
				AdminUser: core.User{
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccessKey,
				},
				Policies: map[string]*iampol.IAMPolicy{},
				Bindings: []*core.PolicyBinding{},
				Users:    map[string]*core.User{},
			}

			err := yaml.MarshalToFile(cfg, managementConfigPath)
			if err != nil {
				return err
			}

			b.Logger.Info("YAML management backend initialized with admin credentials",
				"AWS_ACCESS_KEY_ID", accessKeyID, "AWS_SECRET_ACCESS_KEY", secretAccessKey)

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

func (b *Backend) CreateUser(ctx context.Context, username string) (*core.User, error) {
	if username == "" {
		return nil, core.ErrUserInvalid
	}

	accessKeyID, secretAccessKey := credentials.GenerateCredentials()
	newUser := &core.User{
		Name:            username,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}

	err := b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		if _, ok := cfg.Users[username]; ok {
			return cfg, core.ErrUserAlreadyExists
		}

		cfg.Users[username] = newUser

		return cfg, nil
	})
	if err != nil {
		return nil, err
	}

	return newUser, nil
}

func (b *Backend) UpdateUser(ctx context.Context, updatedUser *core.User) error {
	if updatedUser.Name == "" || updatedUser.AccessKeyID == "" || updatedUser.SecretAccessKey == "" {
		return core.ErrUserInvalid
	}

	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		if _, ok := cfg.Users[updatedUser.Name]; !ok {
			return cfg, core.ErrUserNotFound
		}

		cfg.Users[updatedUser.Name] = &core.User{
			Name:            updatedUser.Name,
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

func (b *Backend) GetBindings(_ context.Context) ([]*core.PolicyBinding, error) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	return b.bindings, nil
}

func (b *Backend) GetBindingsByUser(_ context.Context, userName string) ([]*core.PolicyBinding, error) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	bindings, ok := b.bindingsByUser[userName]
	if !ok {
		return nil, nil
	}

	return bindings, nil
}

func (b *Backend) GetBindingsByPolicy(_ context.Context, policyID string) ([]*core.PolicyBinding, error) {
	b.rwLock.RLock()
	defer b.rwLock.RUnlock()

	bindings, ok := b.bindingsByPolicy[policyID]
	if !ok {
		return nil, nil
	}

	return bindings, nil
}

func (b *Backend) CreateBinding(ctx context.Context, binding *core.PolicyBinding) error {
	if binding.UserName == "" || binding.PolicyID == "" {
		return core.ErrBindingInvalid
	}

	b.rwLock.RLock()
	// Validate user exists
	userExists := binding.UserName == b.adminUser.Name || b.usersByName[binding.UserName] != nil
	if !userExists {
		b.rwLock.RUnlock()

		return core.ErrUserNotFound
	}

	// Validate policy exists
	_, policyExists := b.policiesByID[binding.PolicyID]
	if !policyExists {
		b.rwLock.RUnlock()

		return core.ErrPolicyNotFound
	}

	b.rwLock.RUnlock()

	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		// Check if binding already exists
		exists := lo.ContainsBy(cfg.Bindings, func(existingBinding *core.PolicyBinding) bool {
			return existingBinding.UserName == binding.UserName && existingBinding.PolicyID == binding.PolicyID
		})
		if exists {
			return cfg, core.ErrBindingAlreadyExists
		}

		cfg.Bindings = append(cfg.Bindings, binding)

		return cfg, nil
	})
}

func (b *Backend) DeleteBinding(ctx context.Context, binding *core.PolicyBinding) error {
	return b.readWriteConfig(ctx, func(cfg ManagementConfig) (ManagementConfig, error) {
		found := lo.ContainsBy(cfg.Bindings, func(existingBinding *core.PolicyBinding) bool {
			return existingBinding.UserName == binding.UserName && existingBinding.PolicyID == binding.PolicyID
		})
		if !found {
			return cfg, core.ErrBindingNotFound
		}

		cfg.Bindings = lo.Filter(cfg.Bindings, func(existingBinding *core.PolicyBinding, _ int) bool {
			return existingBinding.UserName != binding.UserName || existingBinding.PolicyID != binding.PolicyID
		})

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
	b.bindingsByUser = map[string][]*core.PolicyBinding{}
	b.bindingsByPolicy = map[string][]*core.PolicyBinding{}
	b.lastUpdated = info.ModTime()
	b.adminUser = &managementConfig.AdminUser
	b.adminUser.Name = "admin"

	for userName, user := range managementConfig.Users {
		// Ensure stored user has Name set
		if user.Name == "" {
			user.Name = userName
		}

		b.usersByName[userName] = user
		b.usersByAccessKeyID[user.AccessKeyID] = b.usersByName[userName]
	}

	maps.Copy(b.policiesByID, managementConfig.Policies)
	b.bindings = managementConfig.Bindings

	// Index bindings by user and policy
	for _, binding := range managementConfig.Bindings {
		b.bindingsByUser[binding.UserName] = append(b.bindingsByUser[binding.UserName], binding)
		b.bindingsByPolicy[binding.PolicyID] = append(b.bindingsByPolicy[binding.PolicyID], binding)
	}

	return nil
}
