package yaml

import "github.com/zhulik/pal"

func Provide() pal.ServiceDef {
	return pal.ProvideList()
}
