package middlewares

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/backends/common"
)

func ErrorRenderer() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			switch {
			case errors.Is(err, common.ErrBucketNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, common.ErrObjectNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, common.ErrBucketAlreadyExists) || errors.Is(err, common.ErrObjectAlreadyExists):
				return echo.NewHTTPError(http.StatusConflict, err.Error())
			case errors.Is(err, common.ErrBucketNotEmpty):
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			case err == nil:
				return nil
			default:
				return err
			}
		}
	}
}
