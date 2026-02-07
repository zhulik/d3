package management

import (
	"github.com/zhulik/d3/internal/apis/management/middlewares"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&APIUsers{}),
		pal.Provide(&APIPolicies{}),
		pal.Provide(&Server{}),
		pal.Provide(&Echo{}),
		middlewares.Provide(),
	)
}
