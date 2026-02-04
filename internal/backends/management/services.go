package storage

import (
	"fmt"

	"github.com/zhulik/d3/internal/backends/storage/folder"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func Provide(config *core.Config) pal.ServiceDef {
	switch config.StorageBackend {
	case core.StorageBackendFolder:
		return folder.Provide()
	default:
		panic(fmt.Sprintf("unknown backend: %s", config.StorageBackend))
	}
}
