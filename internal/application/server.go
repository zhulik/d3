package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-cz/devslog"
	managementapi "github.com/zhulik/d3/internal/apis/management"
	"github.com/zhulik/d3/internal/apis/s3"
	managementbackend "github.com/zhulik/d3/internal/backends/management"
	"github.com/zhulik/d3/internal/backends/storage"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/locker"
	"github.com/zhulik/pal"
)

func NewServer(config *core.Config) *pal.Pal {
	var handler slog.Handler

	switch config.Environment {
	case "development":
		handler = devslog.NewHandler(os.Stdout, &devslog.Options{
			MaxSlicePrintSize: 4,
			SortKeys:          true,
			TimeFormat:        "[04:05]",
			NewLineAfterLog:   true,
			DebugColor:        devslog.Magenta,
			StringerFormatter: true,
		})
	case "test":
		handler = slog.DiscardHandler
	default:
		handler = slog.NewJSONHandler(os.Stdout, nil)
	}

	slog.SetDefault(slog.New(handler))

	return pal.New(
		s3.Provide(),
		managementapi.Provide(),
		managementbackend.Provide(config),
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

func RunServer() {
	config := &core.Config{}
	if err := config.Init(context.Background()); err != nil {
		slog.Error("failed to initialize config", "error", err)
		os.Exit(1)
	}

	p := NewServer(config)

	err := p.Run(context.Background())
	if err != nil {
		slog.Error("failed to run application", "error", err)
		os.Exit(1)
	}
}
