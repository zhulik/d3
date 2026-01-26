package application

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/golang-cz/devslog"
	"github.com/zhulik/d3/internal/backends/folder"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/d3/internal/server"
	"github.com/zhulik/pal"
)

func Run() {
	config := &core.Config{}
	if err := config.Init(context.Background()); err != nil {
		slog.Error("failed to initialize config", "error", err)
		os.Exit(1)
	}

	if config.Environment == "development" {
		// new logger with options
		opts := &devslog.Options{
			MaxSlicePrintSize: 4,
			SortKeys:          true,
			TimeFormat:        "[04:05]",
			NewLineAfterLog:   true,
			DebugColor:        devslog.Magenta,
			StringerFormatter: true,
		}

		logger := slog.New(devslog.NewHandler(os.Stdout, opts))
		slog.SetDefault(logger)
	}

	p := pal.New(
		server.Provide(),
		folder.Provide(config),
		pal.Provide(config),
		locker.Provide(),
	).
		InitTimeout(1*time.Minute).
		HealthCheckTimeout(5*time.Second).
		ShutdownTimeout(1*time.Minute).
		InjectSlog().
		RunHealthCheckServer("0.0.0.0:8081", "/healthz")

	err := p.Run(context.Background())
	if err != nil {
		slog.Error("failed to run application", "error", err)
		os.Exit(1)
	}
}
