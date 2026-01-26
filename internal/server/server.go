package server

import (
	"context"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
)

type Server struct {
	Backend core.Backend

	ObjectsAPI *APIObjects
	BucketsAPI *APIBuckets

	Echo *Echo
}

func (s *Server) Init(ctx context.Context) error {
	buckets := s.Echo.Group("/:bucket")

	buckets.GET("", s.Echo.rootQueryRouter.Handle)

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	sc := echo.StartConfig{Address: ":8080"}
	if err := sc.Start(ctx, s.Echo.Echo); err != nil {
		return err
	}
	return nil
}
