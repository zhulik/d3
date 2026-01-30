package apictx

import (
	"github.com/labstack/echo/v5"
)

// Middleware is an Echo middleware that injects ApiCtx into the request context.
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Inject ApiCtx into the request context
			ctx := Inject(c)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
