package application

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/golang-cz/devslog"
	"github.com/zhulik/d3/internal/client"
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func NewClient(config *core.ClientConfig) *pal.Pal {
	opts := &devslog.Options{
		HandlerOptions: &slog.HandlerOptions{
			Level: slog.LevelError,
		},

		MaxSlicePrintSize: 4,
		SortKeys:          true,
		TimeFormat:        "[04:05]",
		NewLineAfterLog:   true,
		DebugColor:        devslog.Magenta,
		StringerFormatter: true,
	}

	h := devslog.NewHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(h))
	slog.SetLogLoggerLevel(slog.LevelError)

	return pal.New(
		pal.Provide(config),
		apiclient.Provide(),
		client.Provide(),
	).
		InitTimeout(1 * time.Minute).
		HealthCheckTimeout(5 * time.Second).
		ShutdownTimeout(1 * time.Minute).
		InjectSlog()
}

func RunClient() {
	config := &core.ClientConfig{}
	if err := config.Init(context.Background()); err != nil {
		slog.Error("failed to initialize config", "error", err)
		os.Exit(1)
	}

	p := NewClient(config)

	err := p.Run(context.Background())
	if err != nil {
		slog.Error("failed to run application", "error", err)
		os.Exit(1)
	}
}
