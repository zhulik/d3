package folder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/yaml"
)

const (
	pollInterval = 5 * time.Second
)

type UserRepository struct {
	Config *Config
	Logger *slog.Logger

	Backend core.Backend // to make sure we init the backend first

	lastUpdated        time.Time
	adminUser          core.User
	usersByName        map[string]core.User
	usersByAccessKeyID map[string]core.User

	rwLock sync.RWMutex
}

func (r *UserRepository) Init(ctx context.Context) error {
	return r.reload(ctx)
}

func (r *UserRepository) GetUserByName(_ context.Context, name string) (*core.User, error) {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()

	if name == r.adminUser.Name {
		return &r.adminUser, nil
	}

	user, ok := r.usersByName[name]
	if !ok {
		return nil, common.ErrUserNotFound
	}

	return &user, nil
}

func (r *UserRepository) GetUserByAccessKeyID(_ context.Context, accessKeyID string) (*core.User, error) {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()

	if accessKeyID == r.adminUser.AccessKeyID {
		return &r.adminUser, nil
	}

	user, ok := r.usersByAccessKeyID[accessKeyID]
	if !ok {
		return nil, common.ErrUserNotFound
	}

	return &user, nil
}

func (r *UserRepository) CreateUser(_ context.Context, _ core.User) error {
	panic("not implemented")
}

func (r *UserRepository) DeleteUser(_ context.Context, _ string) error {
	panic("not implemented")
}

func (r *UserRepository) AdminCredentials() (string, string) {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()

	return r.adminUser.AccessKeyID, r.adminUser.SecretAccessKey
}

// Run watches the main config file and reloads the user repository when it changes.
func (r *UserRepository) Run(ctx context.Context) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	errorsCount := 0

	var allErrors error

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := r.checkAndReload(ctx)
			if err != nil {
				allErrors = errors.Join(allErrors, err)
				r.Logger.Error("failed to check and reload user repository", "error", err)
			}

			if errorsCount > 3 {
				return fmt.Errorf("failed to check and reload user repository after 3 attempts: %w", allErrors)
			}
		}
	}
}

func (r *UserRepository) checkAndReload(ctx context.Context) error {
	path := r.Config.configYamlPath()

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	r.rwLock.RLock()

	if r.lastUpdated.IsZero() || info.ModTime() != r.lastUpdated {
		r.rwLock.RUnlock()

		r.Logger.Info("config file changed, reloading user repository")

		return r.reload(ctx)
	}

	r.rwLock.RUnlock()

	return nil
}

func (r *UserRepository) reload(ctx context.Context) error {
	r.rwLock.Lock()
	defer r.rwLock.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	path := r.Config.configYamlPath()

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	configYaml, err := yaml.UnmarshalFromFile[configYaml](path)
	if err != nil {
		return err
	}

	r.usersByName = map[string]core.User{}
	r.usersByAccessKeyID = map[string]core.User{}
	r.lastUpdated = info.ModTime()
	r.adminUser = configYaml.AdminUser.toCoreUser()

	for _, user := range configYaml.Users {
		r.usersByName[user.Name] = user.toCoreUser()
		r.usersByAccessKeyID[user.AccessKeyID] = r.usersByName[user.Name]
	}

	return nil
}
