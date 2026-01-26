package server

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
	ihttp "github.com/zhulik/d3/internal/http"
)

type APIBuckets struct {
	Backend core.Backend

	Echo *Echo
}

type bucketsResult struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListAllMyBucketsResult"`
	Owner   *types.Owner
	Buckets []*types.Bucket `xml:"Buckets>Bucket"`
}

type locationConstraintResponse struct {
	XMLName  xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ LocationConstraint"`
	Location string   `xml:",chardata"`
}

type prefixEntry struct {
	Prefix string `xml:"Prefix"`
}

type listObjectsResult struct {
	IsTruncated    bool            `xml:"IsTruncated"`
	Contents       []*types.Object `xml:"Contents"`
	Name           string          `xml:"Name"`
	Prefix         string          `xml:"Prefix"`
	Delimiter      string          `xml:"Delimiter,omitempty"`
	MaxKeys        int             `xml:"MaxKeys"`
	CommonPrefixes []prefixEntry   `xml:"CommonPrefixes,omitempty"`
}

func (a APIBuckets) Init(_ context.Context) error {
	a.Echo.AddQueryParamRoute("location", a.GetBucketLocation)

	a.Echo.GET("/", a.ListBuckets)

	buckets := a.Echo.Group("/:bucket")
	buckets.HEAD("", a.HeadBucket)
	buckets.PUT("", a.CreateBucket)
	buckets.DELETE("", a.DeleteBucket)

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

	ihttp.SetHeaders(c, map[string]string{
		"Location":         fmt.Sprintf("/%s", name),
		"x-amz-bucket-arn": fmt.Sprintf("arn:aws:s3:::%s", name),
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
