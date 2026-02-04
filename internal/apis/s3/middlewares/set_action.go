package middlewares

import (
	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/apis/s3/actions"
)

func SetAction(action actions.Action) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			apicCtx := apictx.FromContext(c.Request().Context())
			apicCtx.Action = action

			return next(c)
		}
	}
}
