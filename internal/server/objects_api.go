package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
)

type ObjectsAPI struct {
	Backend core.Backend
}

func (a ObjectsAPI) HeadObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	result, err := a.Backend.HeadObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Last-Modified", result.LastModified.Format(http.TimeFormat))
	c.Response().Header().Set("Content-Length", strconv.FormatInt(result.ContentLength, 10))

	return c.NoContent(http.StatusOK)
}

func (a ObjectsAPI) PutObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	err := a.Backend.PutObject(c.Request().Context(), bucket, key, c.Request().Body)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (a ObjectsAPI) GetObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	contents, err := a.Backend.GetObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	defer contents.Close() //nolint:errcheck

	c.Response().Header().Set("Last-Modified", contents.LastModified.Format(http.TimeFormat))
	c.Response().Header().Set("Content-Length", strconv.FormatInt(contents.Size, 10))
	c.Response().Header().Set("ETag", fmt.Sprintf("%x", "foo"))

	return c.Stream(http.StatusOK, "application/octet-stream", contents)
}

func (a ObjectsAPI) ListObjects(c *echo.Context) error {
	bucket := c.Param("bucket")
	prefix := c.QueryParam("prefix")
	objects, err := a.Backend.ListObjects(c.Request().Context(), bucket, prefix)
	if err != nil {
		return err
	}
	xmlResponse := ListBucketResult{
		IsTruncated:    false,
		Contents:       objects,
		Name:           bucket,
		Prefix:         prefix,
		Delimiter:      c.QueryParam("delimiter"),
		MaxKeys:        1000,
		CommonPrefixes: []PrefixEntry{},
	}
	return c.XML(http.StatusOK, xmlResponse)
}

func (a ObjectsAPI) DeleteObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	err := a.Backend.DeleteObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
