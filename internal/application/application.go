package application

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/zhulik/d3/internal/backends/folder"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/server"
	"github.com/zhulik/pal"
)

func Run() {
	config := &core.Config{}
	if err := config.Init(context.Background()); err != nil {
		slog.Error("failed to initialize config", "error", err)
		os.Exit(1)
	}

	p := pal.New(
		server.Provide(),
		folder.Provide(config),
		pal.Provide(config),
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
