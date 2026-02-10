package middlewares

import (
	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
)

type Authorizer struct {
	Authorizer core.Authorizer
}

func (a *Authorizer) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			return next(c)
		}
	}
}
