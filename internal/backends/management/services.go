package management

import (
	"fmt"

	"github.com/zhulik/d3/internal/backends/management/yaml"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func Provide(config *core.Config) pal.ServiceDef {
	switch config.ManagementBackend {
	case core.ManagementBackendYAML:
		return yaml.Provide()
	default:
		panic(fmt.Sprintf("unknown management backend: %s", config.ManagementBackend))
	}
}
