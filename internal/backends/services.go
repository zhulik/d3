package backends

import (
	"fmt"

	"github.com/zhulik/d3/internal/backends/folder"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func Provide(config *core.Config) pal.ServiceDef {
	switch config.Backend {
	case core.BackendFolder:
		return folder.Provide()
	default:
		panic(fmt.Sprintf("unknown backend: %s", config.Backend))
	}
}
