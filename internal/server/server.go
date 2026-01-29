package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
)

type Server struct {
	Echo   *Echo
	Config *core.Config
	Logger *slog.Logger
}

func (s *Server) Init(_ context.Context) error {
	buckets := s.Echo.Group("/:bucket")

	buckets.GET("", s.Echo.rootQueryRouter.Handle)

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	s.Logger.Info("starting server", "address", fmt.Sprintf(":%d", s.Config.Port))

	address := fmt.Sprintf(":%d", s.Config.Port)
	sc := echo.StartConfig{Address: address}
	if err := sc.Start(ctx, s.Echo.Echo); err != nil {
		return err
	}
	return nil
}
