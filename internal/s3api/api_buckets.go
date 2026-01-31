package s3api

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/s3api/actions"
	"github.com/zhulik/d3/internal/s3api/middlewares"
)

type APIBuckets struct {
	Backend core.Backend

	Echo *Echo
}

func (a APIBuckets) Init(_ context.Context) error {
	a.Echo.AddQueryParamRoute("location", a.GetBucketLocation, actions.GetBucketLocation)

	a.Echo.GET("/", a.ListBuckets, middlewares.SetAction(actions.ListBuckets))

	buckets := a.Echo.Group("/:bucket")
	buckets.HEAD("", a.HeadBucket, middlewares.SetAction(actions.HeadBucket))
	buckets.PUT("", a.CreateBucket, middlewares.SetAction(actions.CreateBucket))
	buckets.DELETE("", a.DeleteBucket, middlewares.SetAction(actions.DeleteBucket))

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
		Buckets: buckets,
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

	return c.NoContent(http.StatusCreated)
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
	bucket := c.Param("bucket")

	err := a.Backend.HeadBucket(c.Request().Context(), bucket)
	if err != nil {
		return err
	}

	response := locationConstraintResponse{
		Location: "local",
	}

	return c.XML(http.StatusOK, response)
}

func (a APIBuckets) HeadBucket(c *echo.Context) error {
	err := a.Backend.HeadBucket(c.Request().Context(), c.Param("bucket"))
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}
