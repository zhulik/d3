package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
)

type Server struct {
	Logger  *slog.Logger
	Backend core.Backend

	e  *echo.Echo
	sc *echo.StartConfig
}

func (s *Server) Init(ctx context.Context) error {
	s.e = echo.New()
	s.sc = &echo.StartConfig{Address: ":8080"}

	s.e.Pre(middleware.RemoveTrailingSlash())
	s.e.Use(middleware.RequestLogger())
	s.e.Use(middleware.Recover())

	s.e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			switch {
			case errors.Is(err, common.ErrBucketNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, common.ErrObjectNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, common.ErrBucketAlreadyExists):
				return echo.NewHTTPError(http.StatusConflict, err.Error())
			case err == nil:
				return nil
			default:
				return err
			}
		}
	})

	s.e.GET("/", s.ListBuckets)

	buckets := s.e.Group("/:bucket")
	buckets.GET("", s.GetBucketLocation)
	buckets.HEAD("", s.HeadBucket)
	buckets.PUT("", s.CreateBucket)
	buckets.DELETE("", s.DeleteBucket)

	objects := buckets.Group("/*")
	objects.HEAD("", s.HeadObject)
	objects.PUT("", s.PutObject)
	objects.GET("", s.GetObject)

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.sc.Start(ctx, s.e); err != nil {
		s.Logger.Error("failed to start server", "error", err)
		return err
	}
	return nil
}
