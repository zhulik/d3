package s3

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/labstack/echo/v5"
	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/apis/s3/actions"
	middlewares2 "github.com/zhulik/d3/internal/apis/s3/middlewares"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/rangeparser"
	"github.com/zhulik/d3/pkg/smartio"
)

type APIObjects struct {
	Backend      core.StorageBackend
	BucketFinder *middlewares2.BucketFinder
	ObjectFinder *middlewares2.ObjectFinder
	Echo         *Echo
}

func (a APIObjects) Init(_ context.Context) error {
	bucketFinder := a.BucketFinder.Middleware()
	objectFinder := a.ObjectFinder.Middleware()

	a.Echo.AddQueryParamRoute("prefix", a.ListObjectsV2, actions.ListObjectsV2, bucketFinder)
	a.Echo.AddQueryParamRoute("list-type", a.ListObjectsV2, actions.ListObjectsV2, bucketFinder)

	objects := a.Echo.Group("/:bucket/*")
	objects.HEAD("", a.HeadObject, middlewares2.SetAction(actions.HeadObject), bucketFinder, objectFinder)
	objects.PUT("", NewQueryParamsRouter().
		SetFallbackHandler(a.PutObject, actions.PutObject, bucketFinder).
		AddRoute("uploadId", a.UploadPart, actions.UploadPart, bucketFinder).
		Handle)

	objects.POST("", NewQueryParamsRouter().
		AddRoute("uploads", a.CreateMultipartUpload, actions.CreateMultipartUpload, bucketFinder).
		AddRoute("uploadId", a.CompleteMultipartUpload, actions.CompleteMultipartUpload, bucketFinder).
		Handle)

	objects.GET("",
		NewQueryParamsRouter().
			SetFallbackHandler(a.GetObject, actions.GetObject, bucketFinder, objectFinder).
			AddRoute("tagging", a.GetObjectTagging, actions.GetObjectTagging, bucketFinder, objectFinder).
			Handle,
	)

	objects.DELETE("", NewQueryParamsRouter().
		SetFallbackHandler(a.DeleteObject, actions.DeleteObject, bucketFinder).
		AddRoute("uploadId", a.AbortMultipartUpload, actions.AbortMultipartUpload, bucketFinder).
		Handle)

	a.Echo.POST("/:bucket",
		NewQueryParamsRouter().
			AddRoute("delete", a.DeleteObjects, actions.DeleteObjects, bucketFinder).
			Handle,
	)

	return nil
}

func (a APIObjects) HeadObject(c *echo.Context) error {
	object := apictx.FromContext(c.Request().Context()).Object

	setObjectHeaders(c, object.Metadata())

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) PutObject(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	key := c.Param("*")

	tags, err := parseTags(c.Request().Header.Get("X-Amz-Tagging"))
	if err != nil {
		return err
	}

	err = bucket.PutObject(c.Request().Context(), key, core.PutObjectInput{
		Reader: c.Request().Body,
		Metadata: core.ObjectMetadata{
			ContentType: c.Request().Header.Get("Content-Type"),
			SHA256:      c.Request().Header.Get("X-Amz-Content-Sha256"),
			Size:        c.Request().ContentLength,
			Tags:        tags,
			Meta:        parseMeta(c),
		},
	})
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) GetObjectTagging(c *echo.Context) error {
	object := apictx.FromContext(c.Request().Context()).Object
	tags := object.Metadata().Tags

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
	object := apictx.FromContext(c.Request().Context()).Object

	var reader io.Reader = object

	metadata := object.Metadata()

	if rangeHeader := c.Request().Header.Get("Range"); rangeHeader != "" {
		parsedRange, err := rangeparser.Parse(rangeHeader, metadata.Size)
		if err != nil {
			return err
		}

		reader, err = smartio.NewRangedReader(object, parsedRange.Start, parsedRange.End)
		if err != nil {
			return err
		}

		metadata.Size = parsedRange.End - parsedRange.Start + 1
		metadata.SHA256Base64 = ""
		SetHeaders(c, map[string]string{
			"Accept-Ranges": "bytes",
			"Content-Range": fmt.Sprintf("bytes %d-%d/%d", parsedRange.Start, parsedRange.End, metadata.Size),
		})
	}

	setObjectHeaders(c, metadata)

	return c.Stream(http.StatusOK, metadata.ContentType, reader)
}

func (a APIObjects) ListObjectsV2(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	prefix := c.QueryParam("prefix")
	listType := c.QueryParam("list-type")
	maxKeys := c.QueryParam("max-keys")

	maxKeysInt := core.MaxKeys

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

	objects, err := bucket.ListObjectsV2(c.Request().Context(), core.ListObjectsV2Input{
		MaxKeys: maxKeysInt,
		Prefix:  prefix,
	})
	if err != nil {
		return err
	}

	xmlResponse := listObjectsV2Result{
		IsTruncated: false,
		Contents: lo.Map(objects, func(object core.Object, _ int) *types.Object {
			return &types.Object{
				Key:          lo.ToPtr(object.Key()),
				LastModified: lo.ToPtr(object.LastModified()),
				Size:         lo.ToPtr(object.Size()),
			}
		}),
		Name:           bucket.Name(),
		Prefix:         prefix,
		Delimiter:      core.Delimiter,
		MaxKeys:        maxKeysInt,
		CommonPrefixes: []prefixEntry{},
	}

	return c.XML(http.StatusOK, xmlResponse)
}

func (a APIObjects) DeleteObject(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	key := c.Param("*")

	results, err := bucket.DeleteObjects(c.Request().Context(), false, key)
	if err != nil {
		return err
	}

	if results[0].Error != nil {
		return results[0].Error
	}

	return c.NoContent(http.StatusNoContent)
}

func (a APIObjects) DeleteObjects(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket

	var deleteReq deleteRequestXML
	if err := xml.NewDecoder(c.Request().Body).Decode(&deleteReq); err != nil {
		return err
	}

	if len(deleteReq.Objects) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no objects specified")
	}

	if len(deleteReq.Objects) > 1000 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many objects specified")
	}

	keys := lo.Map(deleteReq.Objects, func(obj deleteObjectXML, _ int) string {
		return obj.Key
	})
	quiet := deleteReq.Quiet != nil && *deleteReq.Quiet

	results, err := bucket.DeleteObjects(c.Request().Context(), quiet, keys...)
	if err != nil {
		return err
	}

	response := deleteResultXML{
		Deleted: []deletedEntryXML{},
		Errors:  []errorEntryXML{},
	}

	for _, result := range results {
		if result.Error != nil {
			errorCode := "InternalError"
			errorMessage := result.Error.Error()

			if errors.Is(result.Error, core.ErrObjectNotFound) {
				errorCode = "NoSuchKey"
			}

			response.Errors = append(response.Errors, errorEntryXML{
				Code:    errorCode,
				Key:     result.Key,
				Message: errorMessage,
			})
		} else if !quiet {
			response.Deleted = append(response.Deleted, deletedEntryXML{
				Key: result.Key,
			})
		}
	}

	return c.XML(http.StatusOK, response)
}

func (a APIObjects) CreateMultipartUpload(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	key := c.Param("*")

	tags, err := parseTags(c.Request().Header.Get("X-Amz-Tagging"))
	if err != nil {
		return err
	}

	uploadID, err := bucket.CreateMultipartUpload(c.Request().Context(), key, core.ObjectMetadata{
		ContentType:  c.Request().Header.Get("Content-Type"),
		Tags:         tags,
		LastModified: time.Now(),
		Meta:         parseMeta(c),
	})
	if err != nil {
		return err
	}

	response := initiateMultipartUploadResultXML{
		Bucket:   bucket.Name(),
		Key:      key,
		UploadID: uploadID,
	}

	return c.XML(http.StatusOK, response)
}

func (a APIObjects) UploadPart(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	key := c.Param("*")
	uploadID := c.QueryParam("uploadId")
	partNumber := c.QueryParam("partNumber")

	partNumberInt, err := strconv.Atoi(partNumber)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid part number")
	}

	err = bucket.UploadPart(c.Request().Context(), key, uploadID, partNumberInt, c.Request().Body)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) CompleteMultipartUpload(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	key := c.Param("*")
	uploadID := c.QueryParam("uploadId")

	var req completeMultipartUploadRequestXML
	if err := xml.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid XML body")
	}

	if len(req.Parts) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no parts specified")
	}

	parts := lo.Map(req.Parts, func(part partXML, _ int) core.CompletePart {
		return core.CompletePart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		}
	})

	err := bucket.CompleteMultipartUpload(c.Request().Context(), key, uploadID, parts)
	if err != nil {
		return err
	}

	response := completeMultipartUploadResultXML{
		Bucket: bucket.Name(),
		Key:    key,
	}

	return c.XML(http.StatusOK, response)
}

func (a APIObjects) AbortMultipartUpload(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	key := c.Param("*")
	uploadID := c.QueryParam("uploadId")

	err := bucket.AbortMultipartUpload(c.Request().Context(), key, uploadID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func parseTags(header string) (map[string]string, error) {
	if header == "" {
		return nil, nil //nolint:nilnil
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
	SetHeaders(c, headers)
}

func SetHeaders(c *echo.Context, headers map[string]string) {
	for key, value := range headers {
		c.Response().Header().Set(key, value)
	}
}
