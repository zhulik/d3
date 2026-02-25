package s3

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/labstack/echo/v5"
	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/apis/s3/middlewares"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/s3actions"
)

type APIBuckets struct {
	Backend core.StorageBackend

	BucketFinder *middlewares.BucketFinder
	Echo         *Echo
}

func (a APIBuckets) Init(_ context.Context) error {
	bucketFinder := a.BucketFinder.Middleware()
	authorizer := a.Echo.Authorizer.Middleware()

	a.Echo.AddQueryParamRoute("location", a.GetBucketLocation, s3actions.GetBucketLocation, bucketFinder, authorizer)

	a.Echo.GET("/", a.ListBuckets, middlewares.SetAction(s3actions.ListBuckets), authorizer)

	buckets := a.Echo.Group("/:bucket")
	buckets.HEAD("", a.HeadBucket, middlewares.SetAction(s3actions.HeadBucket), bucketFinder, authorizer)
	buckets.PUT("", a.CreateBucket, middlewares.SetAction(s3actions.CreateBucket), authorizer)
	buckets.DELETE("", a.DeleteBucket, middlewares.SetAction(s3actions.DeleteBucket), authorizer)

	return nil
}

// ListBuckets enumerates existing JetStream Object Store buckets and returns
// a simple S3-compatible XML response.
func (a APIBuckets) ListBuckets(c *echo.Context) error {
	buckets, err := a.Backend.ListBuckets(c.Request().Context())
	if err != nil {
		return err
	}

	response := bucketsResult{
		Buckets: lo.Map(buckets, func(bucket core.Bucket, _ int) *types.Bucket {
			return &types.Bucket{
				Name:         aws.String(bucket.Name()),
				CreationDate: aws.Time(bucket.CreationDate()),
				BucketRegion: aws.String(bucket.Region()),
				BucketArn:    aws.String(bucket.ARN()),
			}
		}),
	}

	return c.XML(http.StatusOK, response)
}

func (a APIBuckets) CreateBucket(c *echo.Context) error {
	name := c.Param("bucket")

	err := a.Backend.CreateBucket(c.Request().Context(), name)
	if err != nil {
		return err
	}

	SetHeaders(c, map[string]string{
		"Location":         "/" + name,
		"x-amz-bucket-arn": "arn:aws:s3:::" + name,
	})

	return c.NoContent(http.StatusOK)
}

func (a APIBuckets) DeleteBucket(c *echo.Context) error {
	name := c.Param("bucket")

	err := a.Backend.DeleteBucket(c.Request().Context(), name)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (a APIBuckets) GetBucketLocation(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket

	response := locationConstraintResponse{
		Location: bucket.Region(),
	}

	return c.XML(http.StatusOK, response)
}

func (a APIBuckets) HeadBucket(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket

	SetHeaders(c, map[string]string{
		"x-amz-bucket-arn":    bucket.ARN(),
		"x-amz-bucket-region": bucket.Region(),
	})

	return c.NoContent(http.StatusOK)
}
