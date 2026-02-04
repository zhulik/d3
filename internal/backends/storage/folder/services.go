package folder

import (
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide[core.StorageBackend](&Backend{}),
		pal.Provide[core.ManagementBackend](&UserRepository{}),
		pal.Provide(&AtomicWriter{}),
		pal.Provide(&Config{}),
	)
}
