package s3api

import (
	"github.com/zhulik/d3/internal/s3api/middlewares"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Server{}),
		pal.Provide(&APIObjects{}),
		pal.Provide(&APIBuckets{}),
		pal.Provide(&Echo{}),
		middlewares.Provide(),
	)
}
