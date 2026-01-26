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
	ihttp "github.com/zhulik/d3/internal/http"
)

type Server struct {
	Logger  *slog.Logger
	Backend core.Backend

	ObjectsAPI *ObjectsAPI
	BucketsAPI *BucketsAPI

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

	s.e.GET("/", s.BucketsAPI.ListBuckets)

	buckets := s.e.Group("/:bucket")

	buckets.GET("", ihttp.NewQueryParamsRouter().
		AddRoute("location", s.BucketsAPI.GetBucketLocation).
		AddRoute("prefix", s.ObjectsAPI.ListObjects).
		Handle,
	)
	buckets.HEAD("", s.BucketsAPI.HeadBucket)
	buckets.PUT("", s.BucketsAPI.CreateBucket)
	buckets.DELETE("", s.BucketsAPI.DeleteBucket)

	objects := buckets.Group("/*")
	objects.HEAD("", s.ObjectsAPI.HeadObject)
	objects.PUT("", s.ObjectsAPI.PutObject)
	objects.GET("", s.ObjectsAPI.GetObject)
	objects.DELETE("", s.ObjectsAPI.DeleteObject)

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.sc.Start(ctx, s.e); err != nil {
		s.Logger.Error("failed to start server", "error", err)
		return err
	}
	return nil
}
