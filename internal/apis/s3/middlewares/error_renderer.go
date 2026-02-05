package middlewares

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
)

func ErrorRenderer() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			switch {
			case errors.Is(err, core.ErrBucketNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, core.ErrObjectNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, core.ErrBucketAlreadyExists) || errors.Is(err, core.ErrObjectAlreadyExists):
				return echo.NewHTTPError(http.StatusConflict, err.Error())
			case errors.Is(err, core.ErrBucketNotEmpty):
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			case err == nil:
				return nil
			default:
				return err
			}
		}
	}
}
