package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-cz/devslog"
	"github.com/zhulik/d3/internal/backends/storage"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/internal/managementapi"
	"github.com/zhulik/d3/internal/s3api"
	"github.com/zhulik/pal"
)

func New(config *core.Config) *pal.Pal {
	var logger *slog.Logger

	if config.Environment == "development" ||
		config.Environment == "test" {
		opts := &devslog.Options{
			MaxSlicePrintSize: 4,
			SortKeys:          true,
			TimeFormat:        "[04:05]",
			NewLineAfterLog:   true,
			DebugColor:        devslog.Magenta,
			StringerFormatter: true,
		}

		logger = slog.New(devslog.NewHandler(os.Stdout, opts))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	slog.SetDefault(logger)

	return pal.New(
		s3api.Provide(),
		managementapi.Provide(),
		storage.Provide(config),
		pal.Provide(config),
		locker.Provide(),
	).
		InitTimeout(1*time.Minute).
		HealthCheckTimeout(5*time.Second).
		ShutdownTimeout(1*time.Minute).
		InjectSlog().
		RunHealthCheckServer(fmt.Sprintf("0.0.0.0:%d", config.HealthCheckPort), "/healthz")
}

func Run() {
	config := &core.Config{}
	if err := config.Init(context.Background()); err != nil {
		slog.Error("failed to initialize config", "error", err)
		os.Exit(1)
	}

	p := New(config)

	err := p.Run(context.Background())
	if err != nil {
		slog.Error("failed to run application", "error", err)
		os.Exit(1)
	}
}
