package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/samber/lo"
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

	setObjectHeaders(c, result)

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) PutObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	tags, err := parseTags(c.Request().Header.Get("x-amz-tagging"))
	if err != nil {
		return err
	}

	meta := parseMeta(c)

	err = a.Backend.PutObject(c.Request().Context(), bucket, key, core.PutObjectInput{
		Reader: c.Request().Body,
		Metadata: core.ObjectMetadata{
			ContentType: c.Request().Header.Get("Content-Type"),
			SHA256:      c.Request().Header.Get("x-amz-content-sha256"),
			Size:        c.Request().ContentLength,
			Tags:        tags,
			Meta:        meta,
		},
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

	setObjectHeaders(c, contents.ObjectMetadata)

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

func parseTags(header string) (map[string]string, error) {
	if header == "" {
		return nil, nil
	}

	decoded, err := url.QueryUnescape(header)
	if err != nil {
		return nil, err
	}

	values, err := url.ParseQuery(decoded)
	if err != nil {
		return nil, err
	}

	tags := map[string]string{}
	for key, vals := range values {
		if len(vals) > 0 {
			tags[key] = vals[0]
		}
	}

	return tags, nil
}

func encodeTags(tags map[string]string) string {
	values := url.Values{}
	for key, value := range tags {
		values.Add(key, value)
	}
	return values.Encode()
}

func parseMeta(c *echo.Context) map[string]string {
	meta := map[string]string{}
	for name, vals := range c.Request().Header {
		ln := strings.ToLower(name)
		// Extract user metadata (x-amz-meta-*)
		if strings.HasPrefix(ln, "x-amz-meta-") {
			meta[ln] = strings.Join(vals, ",")
		}
		// Extract object retention headers if present during PUT
		if ln == "x-amz-object-lock-mode" || ln == "x-amz-object-lock-retain-until-date" {
			meta[ln] = strings.Join(vals, ",")
		}
	}
	return meta
}

func setObjectHeaders(c *echo.Context, metadata *core.ObjectMetadata) {
	headers := lo.Assign(metadata.Meta, map[string]string{
		"Last-Modified":         metadata.LastModified.Format(http.TimeFormat),
		"Content-Length":        strconv.FormatInt(metadata.Size, 10),
		"Content-Type":          metadata.ContentType,
		"ETag":                  metadata.SHA256,
		"x-amz-checksum-sha256": metadata.SHA256Base64,
		"x-amz-tagging":         encodeTags(metadata.Tags),
	})
	ihttp.SetHeaders(c, headers)
}
