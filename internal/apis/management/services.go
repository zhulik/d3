package management

import (
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&APIUsers{}),
		pal.Provide(&Server{}),
		pal.Provide(&Echo{}),
	)
}
