package s3

import (
	"github.com/zhulik/d3/internal/apis/s3/auth"
	"github.com/zhulik/d3/internal/apis/s3/middlewares"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Server{}),
		pal.Provide(&APIObjects{}),
		pal.Provide(&APIBuckets{}),
		pal.Provide(&Echo{}),
		auth.Provide(),
		middlewares.Provide(),
	)
}
