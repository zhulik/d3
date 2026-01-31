package s3api

import "github.com/zhulik/pal"

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Server{}),
		pal.Provide(&APIObjects{}),
		pal.Provide(&APIBuckets{}),
		pal.Provide(&Echo{}),
	)
}
