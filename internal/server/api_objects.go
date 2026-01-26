package server

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
	ihttp "github.com/zhulik/d3/internal/http"
)

type APIObjects struct {
	Logger *slog.Logger

	Backend core.Backend
	Echo    *Echo
}

func (a APIObjects) Init(_ context.Context) error {
	a.Echo.AddQueryParamRoute("prefix", a.ListObjects)
	a.Echo.AddQueryParamRoute("list-type", a.ListObjects)

	objects := a.Echo.Group("/:bucket/*")
	objects.HEAD("", a.HeadObject)
	objects.PUT("", a.PutObject)
	objects.GET("", a.GetObject)
	objects.DELETE("", a.DeleteObject)

	return nil
}

func (a APIObjects) HeadObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	result, err := a.Backend.HeadObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	ihttp.SetHeaders(c, map[string]string{
		"Last-Modified":  result.LastModified.Format(http.TimeFormat),
		"Content-Length": strconv.FormatInt(result.ContentLength, 10),
	})

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) PutObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	a.Logger.Info("put object", "bucket", bucket, "key", key, "headers", c.Request().Header)

	err := a.Backend.PutObject(c.Request().Context(), bucket, key, core.PutObjectInput{
		Reader:      c.Request().Body,
		ContentType: c.Request().Header.Get("Content-Type"),
		SHA256:      c.Request().Header.Get("x-amz-content-sha256"),
		// TODO: metadata
	})
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) GetObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	contents, err := a.Backend.GetObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	defer contents.Close() //nolint:errcheck

	ihttp.SetHeaders(c, map[string]string{
		"Last-Modified":         contents.LastModified.Format(http.TimeFormat),
		"Content-Length":        strconv.FormatInt(contents.Size, 10),
		"Content-Type":          contents.ContentType,
		"ETag":                  contents.SHA256,
		"x-amz-checksum-sha256": contents.SHA256Base64,
		// TODO: metadata
	})

	return c.Stream(http.StatusOK, "application/octet-stream", contents)
}

func (a APIObjects) ListObjects(c *echo.Context) error {
	bucket := c.Param("bucket")
	prefix := c.QueryParam("prefix")

	objects, err := a.Backend.ListObjects(c.Request().Context(), bucket, prefix)
	if err != nil {
		return err
	}

	xmlResponse := listBucketResult{
		IsTruncated:    false,
		Contents:       objects,
		Name:           bucket,
		Prefix:         prefix,
		Delimiter:      c.QueryParam("delimiter"),
		MaxKeys:        1000,
		CommonPrefixes: []prefixEntry{},
	}

	return c.XML(http.StatusOK, xmlResponse)
}

func (a APIObjects) DeleteObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	err := a.Backend.DeleteObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
