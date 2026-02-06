package management

import (
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&APIUsers{}),
		pal.Provide(&APIPolicies{}),
		pal.Provide(&Server{}),
		pal.Provide(&Echo{}),
	)
}
