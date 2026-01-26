package folder

import (
	"log/slog"
)

type Backend struct {
	Logger *slog.Logger
	*BackendBuckets
	*BackendObjects
}
