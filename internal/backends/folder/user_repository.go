package folder

import (
	"context"
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

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := r.checkAndReload(ctx)

			if err != nil {
				r.Logger.Error("failed to check and reload user repository", "error", err)
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

	usersByName := map[string]core.User{}
	usersByAccessKeyID := map[string]core.User{}
	for _, user := range configYaml.Users {
		usersByName[user.Name] = core.User{
			Name:            user.Name,
			AccessKeyID:     user.AccessKeyID,
			SecretAccessKey: user.SecretAccessKey,
		}
		usersByAccessKeyID[user.AccessKeyID] = usersByName[user.Name]
	}

	r.adminUser = core.User{
		Name:            configYaml.AdminUser.Name,
		AccessKeyID:     configYaml.AdminUser.AccessKeyID,
		SecretAccessKey: configYaml.AdminUser.SecretAccessKey,
	}

	r.usersByName = usersByName
	r.usersByAccessKeyID = usersByAccessKeyID

	r.lastUpdated = info.ModTime()
	return nil
}
