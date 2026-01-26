package server

import "github.com/zhulik/pal"

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Server{}),
		pal.Provide(&ObjectsAPI{}),
		pal.Provide(&BucketsAPI{}),
	)
}
