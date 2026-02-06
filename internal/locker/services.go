package locker

import (
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.Provide[core.Locker](&Locker{})
}
