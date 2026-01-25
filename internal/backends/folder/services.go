package folder

import (
	"fmt"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func Provide(config *core.Config) pal.ServiceDef {
	switch config.Backend {
	case core.BackendFolder:
		return pal.Provide[core.Backend](&Backend{})
	default:
		panic(fmt.Sprintf("unknown backend: %s", config.Backend))
	}
}
