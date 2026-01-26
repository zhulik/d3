package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/zhulik/d3/internal/backends/common"
	ihttp "github.com/zhulik/d3/internal/http"
)

type Echo struct {
	*echo.Echo

	rootQueryRouter *ihttp.QueryParamsRouter
}

func (e *Echo) Init(ctx context.Context) error {
	e.Echo = echo.New()
	e.Logger = slog.Default()
	e.rootQueryRouter = ihttp.NewQueryParamsRouter()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
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

	return nil
}
