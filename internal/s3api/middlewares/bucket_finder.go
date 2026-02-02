package middlewares

import (
	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/core"
)

type BucketFinder struct {
	Backend core.Backend
}

func (b *BucketFinder) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			bucketName := c.Param("bucket")
			if bucketName != "" {
				bucket, err := b.Backend.HeadBucket(c.Request().Context(), bucketName)
				if err != nil {
					return err
				}

				apiCtx := apictx.FromContext(c.Request().Context())
				apiCtx.Bucket = bucket
			}

			return next(c)
		}
	}
}
