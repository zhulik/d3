package folder

import (
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide[core.Backend](&Backend{}),
		pal.Provide(&BackendObjects{}),
		pal.Provide(&BackendBuckets{}),
		pal.Provide(&AtomicWriter{}),
		pal.Provide(&Config{}),
	)
}
