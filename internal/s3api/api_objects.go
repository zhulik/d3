package s3api

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
	"github.com/zhulik/d3/internal/backends/common"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/s3api/actions"
	"github.com/zhulik/d3/internal/s3api/middlewares"
	"github.com/zhulik/d3/pkg/rangeparser"
	"github.com/zhulik/d3/pkg/smartio"
)

type APIObjects struct {
	Backend core.Backend
	Echo    *Echo
}

func (a APIObjects) Init(_ context.Context) error {
	a.Echo.AddQueryParamRoute("prefix", a.ListObjectsV2, actions.ListObjectsV2)
	a.Echo.AddQueryParamRoute("list-type", a.ListObjectsV2, actions.ListObjectsV2)

	objects := a.Echo.Group("/:bucket/*")
	objects.HEAD("", a.HeadObject, middlewares.SetAction(actions.HeadObject))
	objects.PUT("", NewQueryParamsRouter().
		SetFallbackHandler(a.PutObject, actions.PutObject).
		AddRoute("uploadId", a.UploadPart, actions.UploadPart).
		Handle)

	objects.POST("", NewQueryParamsRouter().
		AddRoute("uploads", a.CreateMultipartUpload, actions.CreateMultipartUpload).
		AddRoute("uploadId", a.CompleteMultipartUpload, actions.CompleteMultipartUpload).
		Handle)

	objects.GET("",
		NewQueryParamsRouter().
			SetFallbackHandler(a.GetObject, actions.GetObject).
			AddRoute("tagging", a.GetObjectTagging, actions.GetObjectTagging).
			Handle,
	)

	objects.DELETE("", NewQueryParamsRouter().
		SetFallbackHandler(a.DeleteObject, actions.DeleteObject).
		AddRoute("uploadId", a.AbortMultipartUpload, actions.AbortMultipartUpload).
		Handle)

	a.Echo.POST("/:bucket",
		NewQueryParamsRouter().
			AddRoute("delete", a.DeleteObjects, actions.DeleteObjects).
			Handle,
	)

	return nil
}

func (a APIObjects) HeadObject(c *echo.Context) error {
	bucketName := c.Param("bucket")
	key := c.Param("*")

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
	}

	result, err := bucket.HeadObject(c.Request().Context(), key)
	if err != nil {
		return err
	}

	setObjectHeaders(c, result)

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) PutObject(c *echo.Context) error {
	bucketName := c.Param("bucket")
	key := c.Param("*")

	tags, err := parseTags(c.Request().Header.Get("X-Amz-Tagging"))
	if err != nil {
		return err
	}

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
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
	bucketName := c.Param("bucket")
	key := c.Param("*")

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
	}

	tags, err := bucket.GetObjectTagging(c.Request().Context(), key)
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
	bucketName := c.Param("bucket")
	key := c.Param("*")

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
	}

	contents, err := bucket.GetObject(c.Request().Context(), key)
	if err != nil {
		return err
	}
	defer contents.Close()

	var reader io.Reader = contents

	metadata := contents.Metadata()

	if rangeHeader := c.Request().Header.Get("Range"); rangeHeader != "" {
		parsedRange, err := rangeparser.Parse(rangeHeader, metadata.Size)
		if err != nil {
			return err
		}

		reader, err = smartio.NewRangedReader(contents, parsedRange.Start, parsedRange.End)
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
	bucketName := c.Param("bucket")
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

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
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
		Name:           bucketName,
		Prefix:         prefix,
		Delimiter:      common.Delimiter,
		MaxKeys:        maxKeysInt,
		CommonPrefixes: []prefixEntry{},
	}

	return c.XML(http.StatusOK, xmlResponse)
}

func (a APIObjects) DeleteObject(c *echo.Context) error {
	bucketName := c.Param("bucket")
	key := c.Param("*")

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
	}

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
	bucketName := c.Param("bucket")

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

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
	}

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

			if errors.Is(result.Error, common.ErrObjectNotFound) {
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
	bucketName := c.Param("bucket")
	key := c.Param("*")

	tags, err := parseTags(c.Request().Header.Get("X-Amz-Tagging"))
	if err != nil {
		return err
	}

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
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
		Bucket:   bucketName,
		Key:      key,
		UploadID: uploadID,
	}

	return c.XML(http.StatusOK, response)
}

func (a APIObjects) UploadPart(c *echo.Context) error {
	bucketName := c.Param("bucket")
	key := c.Param("*")
	uploadID := c.QueryParam("uploadId")
	partNumber := c.QueryParam("partNumber")

	partNumberInt, err := strconv.Atoi(partNumber)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid part number")
	}

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
	}

	err = bucket.UploadPart(c.Request().Context(), key, uploadID, partNumberInt, c.Request().Body)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) CompleteMultipartUpload(c *echo.Context) error {
	bucketName := c.Param("bucket")
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

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
	}

	err = bucket.CompleteMultipartUpload(c.Request().Context(), key, uploadID, parts)
	if err != nil {
		return err
	}

	response := completeMultipartUploadResultXML{
		Bucket: bucketName,
		Key:    key,
	}

	return c.XML(http.StatusOK, response)
}

func (a APIObjects) AbortMultipartUpload(c *echo.Context) error {
	bucketName := c.Param("bucket")
	key := c.Param("*")
	uploadID := c.QueryParam("uploadId")

	bucket, err := a.Backend.HeadBucket(c.Request().Context(), bucketName)
	if err != nil {
		return err
	}

	err = bucket.AbortMultipartUpload(c.Request().Context(), key, uploadID)
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
