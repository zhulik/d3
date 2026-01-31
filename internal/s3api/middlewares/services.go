package middlewares

import (
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.Provide(&Authenticator{})
}
