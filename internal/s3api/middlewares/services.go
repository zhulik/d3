package middlewares

import (
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Authenticator{}),
		pal.Provide(&BucketFinder{}),
	)
}
