package server

import (
	"context"
	"encoding/xml"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	ihttp "github.com/zhulik/d3/internal/http"
)

type taggingXML struct {
	XMLName xml.Name  `xml:"Tagging"`
	TagSet  tagSetXML `xml:"TagSet"`
}

type tagSetXML struct {
	Tags []tagXML `xml:"Tag"`
}

type tagXML struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type APIObjects struct {
	Logger *slog.Logger

	Backend core.Backend
	Echo    *Echo
}

func (a APIObjects) Init(_ context.Context) error {
	a.Echo.AddQueryParamRoute("prefix", a.ListObjectsV2)
	a.Echo.AddQueryParamRoute("list-type", a.ListObjectsV2)

	objects := a.Echo.Group("/:bucket/*")
	objects.HEAD("", a.HeadObject)
	objects.PUT("", a.PutObject)

	objects.GET("",
		ihttp.NewQueryParamsRouter().
			SetFallbackHandler(a.GetObject).
			AddRoute("tagging", a.GetObjectTagging).
			Handle,
	)

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

func (a APIObjects) GetObjectTagging(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	tags, err := a.Backend.GetObjectTagging(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	tagging := taggingXML{
		TagSet: tagSetXML{
			Tags: make([]tagXML, 0, len(tags)),
		},
	}

	for key, value := range tags {
		tagging.TagSet.Tags = append(tagging.TagSet.Tags, tagXML{
			Key:   key,
			Value: value,
		})
	}

	return c.XML(http.StatusOK, tagging)
}

func (a APIObjects) GetObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	contents, err := a.Backend.GetObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	defer contents.Reader.Close() //nolint:errcheck

	setObjectHeaders(c, contents.Metadata)

	return c.Stream(http.StatusOK, contents.Metadata.ContentType, contents.Reader)
}

func (a APIObjects) ListObjectsV2(c *echo.Context) error {
	bucket := c.Param("bucket")
	prefix := c.QueryParam("prefix")
	listType := c.QueryParam("list-type")
	maxKeys := c.QueryParam("max-keys")

	maxKeysInt := common.MaxKeys
	var err error

	if maxKeys != "" {
		maxKeysInt, err = strconv.Atoi(maxKeys)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid max-keys")
		}
	}

	if listType != "2" {
		return echo.NewHTTPError(http.StatusNotImplemented, "only ListObjectsV2 is supported")
	}

	objects, err := a.Backend.ListObjectsV2(c.Request().Context(), bucket, core.ListObjectsV2Input{
		MaxKeys: maxKeysInt,
		Prefix:  prefix,
	})
	if err != nil {
		return err
	}

	xmlResponse := listObjectsV2Result{
		IsTruncated:    false,
		Contents:       objects,
		Name:           bucket,
		Prefix:         prefix,
		Delimiter:      common.Delimiter,
		MaxKeys:        maxKeysInt,
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

func parseMeta(c *echo.Context) map[string]string {
	meta := map[string]string{}
	for name, vals := range c.Request().Header {
		ln := strings.ToLower(name)
		if strings.HasPrefix(ln, "x-amz-meta-") {
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
		"x-amz-tagging-count":   strconv.Itoa(len(metadata.Tags)),
	})
	ihttp.SetHeaders(c, headers)
}
