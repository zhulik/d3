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

			bucket := apiCtx.Bucket.Name()

			// Decide whether this is a bucket-level or object-level operation.
			var resource string

			switch apiCtx.Action { //nolint:exhaustive
			case s3actions.CreateBucket,
				s3actions.HeadBucket,
				s3actions.DeleteBucket,
				s3actions.GetBucketLocation,
				s3actions.ListObjectsV2:
				// Bucket-level operations.
				resource = bucket
			case s3actions.ListBuckets:
				// Account-level operation – currently allowed without IAM checks.
				return next(c)
			default:
				// Object-level operations. Some (like DeleteObjects) may not have a specific key.
				if apiCtx.Object == nil {
					resource = bucket
				} else {
					resource = bucket + "/" + apiCtx.Object.Key()
				}
			}

			allowed, err := a.Authorizer.IsAllowed(c.Request().Context(), apiCtx.Username, apiCtx.Action, resource)
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
