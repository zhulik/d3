package middlewares

import (
	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/s3actions"
)

// Authorizer is a thin wrapper around core.Authorizer used as an Echo middleware.
// It expects that apictx.Middleware and, for S3 routes, middlewares.SetAction and
// the bucket/object finder middlewares have already populated APICtx.
type Authorizer struct {
	Authorizer core.Authorizer
}

func (a *Authorizer) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			apiCtx := apictx.FromContext(c.Request().Context())
			if apiCtx == nil {
				return core.ErrUnauthorized
			}

			var bucketName string
			if apiCtx.Bucket != nil {
				bucketName = apiCtx.Bucket.Name()
			} else {
				bucketName = c.Param("bucket")
			}

			// Decide whether this is a bucket-level or object-level operation.
			var resource string

			switch apiCtx.Action { //nolint:exhaustive
			case s3actions.ListBuckets:
				// Account-level operation – resource is wildcard.
				resource = "*"
			case s3actions.CreateBucket,
				s3actions.HeadBucket,
				s3actions.DeleteBucket,
				s3actions.GetBucketLocation,
				s3actions.ListObjectsV2:
				// Bucket-level operations.
				resource = bucketName
			default:
				// Object-level operations. Some (like DeleteObjects) may not have a specific key.
				if apiCtx.Object == nil {
					resource = bucketName
				} else {
					resource = bucketName + "/" + apiCtx.Object.Key()
				}
			}

			allowed, err := a.Authorizer.IsAllowed(c.Request().Context(), apiCtx.User, apiCtx.Action, resource)
			if err != nil {
				return err
			}

			if !allowed {
				return core.ErrUnauthorized
			}

			return next(c)
		}
	}
}
