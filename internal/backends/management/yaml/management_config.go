package yaml

import (
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iampol"
)

const (
	ConfigVersion = 1
)

type user struct {
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

func (u user) toCoreUser(userName string) core.User {
	return core.User{
		Name:            userName,
		AccessKeyID:     u.AccessKeyID,
		SecretAccessKey: u.SecretAccessKey,
	}
}

type ManagementConfig struct {
	Version   int                         `yaml:"version"`
	AdminUser user                        `yaml:"admin_user"`
	Users     map[string]user             `yaml:"users"`
	Policies  map[string]iampol.IAMPolicy `yaml:"policies"`
}
