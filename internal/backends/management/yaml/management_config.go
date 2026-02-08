package yaml

import (
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iampol"
)

const (
	ConfigVersion = 1
)

// Use core.User directly for YAML marshaling/unmarshaling. core.User has yaml tags.

type ManagementConfig struct {
	Version   int                          `yaml:"version"`
	AdminUser core.User                    `yaml:"admin_user"`
	Users     map[string]*core.User        `yaml:"users"`
	Policies  map[string]*iampol.IAMPolicy `yaml:"policies"`
}
