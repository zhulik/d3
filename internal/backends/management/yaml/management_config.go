package yaml

import "github.com/zhulik/d3/internal/core"

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

type ManagementConfig struct {
	Version   int    `yaml:"version"`
	AdminUser user   `yaml:"admin_user"`
	Users     []user `yaml:"users"`
}
