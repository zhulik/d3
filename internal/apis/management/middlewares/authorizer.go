package middlewares

import (
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/core"
)

type Authorizer struct {
	Logger *slog.Logger
}

func (a *Authorizer) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			apiCtx := apictx.FromContext(c.Request().Context())
			if apiCtx.Username == nil || *apiCtx.Username != "admin" {
				a.Logger.Warn("Unauthorized access attempt", "username", apiCtx.Username)

				return core.ErrUnauthorized
			}

			return next(c)
		}
	}
}
