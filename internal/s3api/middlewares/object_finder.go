package middlewares

import (
	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/core"
)

type ObjectFinder struct {
	Backend core.Backend
}

func (b *ObjectFinder) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			key := c.Param("*")

			apiCtx := apictx.FromContext(c.Request().Context())

			object, err := apiCtx.Bucket.HeadObject(c.Request().Context(), key)
			if err != nil {
				return err
			}

			apiCtx.Object = object

			return next(c)
		}
	}
}
