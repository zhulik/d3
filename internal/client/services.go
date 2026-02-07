package client

import (
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Runner{}),
		pal.Provide(&apiclient.Client{}),
	)
}
