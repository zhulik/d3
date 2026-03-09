package s3

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/labstack/echo/v5"
	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/apis/s3/middlewares"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/conditionalheaders"
	"github.com/zhulik/d3/pkg/rangeparser"
	"github.com/zhulik/d3/pkg/s3actions"
	"github.com/zhulik/d3/pkg/sigv4"
	"github.com/zhulik/d3/pkg/smartio"
)

const (
	StreamingHMACSHA256   = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
	maxObjectTagCount     = 10
	maxObjectTagKeyLength = 128  // Unicode characters per S3
	maxObjectTagValLength = 256  // Unicode characters per S3
	taggingRequestBodyMax = 1024 // 1 KB for PutObjectTagging XML
)

type APIObjects struct {
	Backend      core.StorageBackend
	BucketFinder *middlewares.BucketFinder
	ObjectFinder *middlewares.ObjectFinder
	Echo         *Echo
}

func (a APIObjects) Init(_ context.Context) error {
	bucketFinder := a.BucketFinder.Middleware()
	objectFinder := a.ObjectFinder.Middleware()
	authorizer := a.Echo.Authorizer.Middleware()
	a.Echo.AddQueryParamRoute("uploads", a.ListMultipartUploads, s3actions.ListMultipartUploads, bucketFinder, authorizer)
	a.Echo.AddQueryParamRoute("prefix", a.ListObjectsV2, s3actions.ListObjectsV2, bucketFinder, authorizer)
	a.Echo.AddQueryParamRoute("list-type", a.ListObjectsV2, s3actions.ListObjectsV2, bucketFinder, authorizer)
	a.Echo.AddQueryParamRoute("marker", a.ListObjectsV2, s3actions.ListObjectsV2, bucketFinder, authorizer)
	a.Echo.SetRootFallbackHandler(a.ListObjectsV2, s3actions.ListObjectsV2, bucketFinder, authorizer)

	objects := a.Echo.Group("/:bucket/*", middlewares.ObjectKeyValidator)
	objects.HEAD("", a.HeadObject, middlewares.SetAction(s3actions.HeadObject), bucketFinder, objectFinder, authorizer)
	objects.PUT("", NewQueryParamsRouter().
		SetFallbackHandler(a.PutObject, s3actions.PutObject, bucketFinder, authorizer).
		AddRoute("tagging", a.PutObjectTagging, s3actions.PutObjectTagging, bucketFinder, objectFinder, authorizer).
		AddRoute("uploadId", a.UploadPart, s3actions.UploadPart,
			bucketFinder, middlewares.UploadIDValidator, authorizer).
		Handle)

	objects.POST("", NewQueryParamsRouter().
		AddRoute("uploads", a.CreateMultipartUpload, s3actions.CreateMultipartUpload, bucketFinder, authorizer).
		AddRoute("uploadId", a.CompleteMultipartUpload, s3actions.CompleteMultipartUpload,
			bucketFinder, middlewares.UploadIDValidator, authorizer).
		Handle)

	objects.GET("",
		NewQueryParamsRouter().
			SetFallbackHandler(a.GetObject, s3actions.GetObject, bucketFinder, objectFinder, authorizer).
			AddRoute("tagging", a.GetObjectTagging, s3actions.GetObjectTagging, bucketFinder, objectFinder, authorizer).
			AddRoute("uploadId", a.ListParts, s3actions.ListParts,
				bucketFinder, middlewares.UploadIDValidator, authorizer).
			Handle,
	)

	objects.DELETE("", NewQueryParamsRouter().
		SetFallbackHandler(a.DeleteObject, s3actions.DeleteObject, bucketFinder, authorizer).
		AddRoute("tagging", a.DeleteObjectTagging, s3actions.DeleteObjectTagging, bucketFinder, objectFinder, authorizer).
		AddRoute("uploadId", a.AbortMultipartUpload, s3actions.AbortMultipartUpload,
			bucketFinder, middlewares.UploadIDValidator, authorizer).
		Handle)

	a.Echo.POST("/:bucket",
		NewQueryParamsRouter().
			AddRoute("delete", a.DeleteObjects, s3actions.DeleteObjects, bucketFinder, authorizer).
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
	if copySource := c.Request().Header.Get("X-Amz-Copy-Source"); copySource != "" {
		return a.CopyObject(c, copySource)
	}

	apiCtx := apictx.FromContext(c.Request().Context())
	bucket := apiCtx.Bucket
	key := c.Param("*")

	tags, err := parseTags(c.Request().Header.Get("X-Amz-Tagging"))
	if err != nil {
		return err
	}

	if err := ValidateTags(tags); err != nil {
		return err
	}

	reader := c.Request().Body
	sha256 := c.Request().Header.Get("X-Amz-Content-Sha256")

	if sha256 == StreamingHMACSHA256 {
		user := apiCtx.User
		authParams := apiCtx.AuthParams

		signer := sigv4.NewChunkSigner(
			authParams.ScopeRegion, authParams.ScopeService,
			authParams.RawSignature(), authParams.RequestTime,
			user.AccessKeyID, user.SecretAccessKey,
		)
		reader = sigv4.NewChunkedReader(reader, signer)
	}

	cond := conditionalheaders.Parse(c.Request().Header)

	ifNoneMatch, condErr := putObjectConditional(c.Request().Context(), bucket, key, cond)
	if condErr != nil {
		return condErr
	}

	err = bucket.PutObject(c.Request().Context(), key, core.PutObjectInput{
		Reader:      reader,
		IfNoneMatch: ifNoneMatch,
		Metadata: core.ObjectMetadata{
			ContentType: c.Request().Header.Get("Content-Type"),
			SHA256:      sha256,
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

// headObjectBucket is the subset of core.Bucket needed for PutObject conditional evaluation.
type headObjectBucket interface {
	HeadObject(ctx context.Context, key string) (core.Object, error)
}

// putObjectConditional evaluates conditional headers for PutObject. It returns (ifNoneMatch, nil) to pass
// to the backend, or (_, err) with core.ErrObjectNotFound (404) or core.ErrPreconditionFailed (412).
func putObjectConditional(
	ctx context.Context, bucket headObjectBucket, key string, cond conditionalheaders.Conditionals,
) (bool, error) {
	hasOther := cond.IfMatch != "" ||
		(cond.IfNoneMatch != "" && cond.IfNoneMatch != "*") ||
		cond.IfModifiedSince != nil || cond.IfUnmodifiedSince != nil

	if !hasOther {
		return cond.IfNoneMatch == "*", nil
	}

	obj, err := bucket.HeadObject(ctx, key)
	if err != nil {
		if errors.Is(err, core.ErrObjectNotFound) {
			if cond.IfMatch != "" {
				return false, core.ErrObjectNotFound
			}

			return cond.IfNoneMatch == "*", nil
		}

		return false, err
	}
	defer obj.Close()

	metadata := obj.Metadata()
	if cond.Check(metadata.SHA256, metadata.LastModified) != http.StatusOK {
		return false, core.ErrPreconditionFailed
	}

	return false, nil
}

func (a APIObjects) CopyObject(c *echo.Context, rawCopySource string) error { //nolint:funlen
	ctx := c.Request().Context()
	apiCtx := apictx.FromContext(ctx)
	dstBucket := apiCtx.Bucket
	dstKey := c.Param("*")

	copySource, err := url.QueryUnescape(rawCopySource)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid x-amz-copy-source")
	}

	copySource = strings.TrimPrefix(copySource, "/")

	srcBucketName, srcKey, ok := strings.Cut(copySource, "/")
	if !ok || srcBucketName == "" || srcKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid x-amz-copy-source")
	}

	if err := core.ValidateObjectKey(srcKey); err != nil {
		return err
	}

	srcBucket, err := a.Backend.HeadBucket(ctx, srcBucketName)
	if err != nil {
		return err
	}

	allowed, err := a.Echo.Authorizer.Authorizer.IsAllowed(ctx, apiCtx.User, s3actions.GetObject, srcBucketName+"/"+srcKey)
	if err != nil {
		return err
	}

	if !allowed {
		return core.ErrUnauthorized
	}

	source, err := srcBucket.GetObject(ctx, srcKey)
	if err != nil {
		return err
	}
	defer source.Close()

	metadataDirective := core.CopyDirective(c.Request().Header.Get("X-Amz-Metadata-Directive"))
	if metadataDirective == "" {
		metadataDirective = core.CopyDirectiveCopy
	}

	taggingDirective := core.CopyDirective(c.Request().Header.Get("X-Amz-Tagging-Directive"))
	if taggingDirective == "" {
		taggingDirective = core.CopyDirectiveCopy
	}

	input := core.CopyObjectInput{
		Source:            source,
		MetadataDirective: metadataDirective,
		TaggingDirective:  taggingDirective,
		IfNoneMatch:       c.Request().Header.Get("If-None-Match") == "*",
	}

	if metadataDirective == core.CopyDirectiveReplace {
		input.ContentType = c.Request().Header.Get("Content-Type")
		input.ReplacementMeta = parseMeta(c)
	}

	if taggingDirective == core.CopyDirectiveReplace {
		input.ReplacementTags, err = parseTags(c.Request().Header.Get("X-Amz-Tagging"))
		if err != nil {
			return err
		}

		if err := ValidateTags(input.ReplacementTags); err != nil {
			return err
		}
	}

	result, err := dstBucket.CopyObject(ctx, dstKey, input)
	if err != nil {
		return err
	}

	return c.XML(http.StatusOK, copyObjectResultXML{
		ETag:         result.Metadata.SHA256,
		LastModified: result.Metadata.LastModified.Format("2006-01-02T15:04:05.000Z"),
	})
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

func (a APIObjects) PutObjectTagging(c *echo.Context) error {
	apiCtx := apictx.FromContext(c.Request().Context())
	bucket := apiCtx.Bucket
	key := c.Param("*")

	var req taggingXML
	if err := xml.NewDecoder(io.LimitReader(c.Request().Body, taggingRequestBodyMax)).Decode(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid XML body")
	}

	tags := make(map[string]string, len(req.TagSet.Tags))
	for _, t := range req.TagSet.Tags {
		tags[t.Key] = t.Value
	}

	if err := ValidateTags(tags); err != nil {
		return err
	}

	if err := bucket.PutObjectTagging(c.Request().Context(), key, tags); err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) DeleteObjectTagging(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	key := c.Param("*")

	if err := bucket.DeleteObjectTagging(c.Request().Context(), key); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (a APIObjects) GetObject(c *echo.Context) error {
	object := apictx.FromContext(c.Request().Context()).Object

	metadata := object.Metadata()

	cond := conditionalheaders.Parse(c.Request().Header)
	switch cond.Check(metadata.SHA256, metadata.LastModified) {
	case http.StatusNotModified:
		return c.NoContent(http.StatusNotModified)
	case http.StatusPreconditionFailed:
		return core.ErrPreconditionFailed
	}

	var reader io.Reader = object

	if rangeHeader := c.Request().Header.Get("Range"); rangeHeader != "" {
		parsedRange, err := rangeparser.Parse(rangeHeader, metadata.Size)
		if err != nil {
			return err
		}

		reader, err = smartio.NewRangedReader(object, parsedRange.Start, parsedRange.End)
		if err != nil {
			return err
		}

		SetHeaders(c, map[string]string{
			"Content-Length": strconv.FormatInt(parsedRange.Length(), 10),
			"Accept-Ranges":  "bytes",
			"Content-Range":  fmt.Sprintf("bytes %d-%d/%d", parsedRange.Start, parsedRange.End, metadata.Size),
		})
	} else {
		setObjectHeaders(c, metadata)
	}

	return c.Stream(http.StatusOK, metadata.ContentType, reader)
}

func mapObjectsToTypes(objects []core.Object) []*types.Object {
	return lo.Map(objects, func(object core.Object, _ int) *types.Object {
		metadata := object.Metadata()

		return &types.Object{
			Key:          lo.ToPtr(object.Key()),
			LastModified: lo.ToPtr(object.LastModified()),
			Size:         lo.ToPtr(object.Size()),
			ETag:         lo.ToPtr(metadata.SHA256),
		}
	})
}

func listObjectsV1Response(
	ctx context.Context, bucket core.Bucket, prefix, delimiter, marker string, maxKeysInt int,
) (listBucketResult, error) {
	v1ContinuationToken := ""
	requestMaxKeys := maxKeysInt

	if marker != "" {
		v1ContinuationToken = base64.StdEncoding.EncodeToString([]byte(marker))
		// Request one extra so that after stripping the marker we still return a full page.
		requestMaxKeys = maxKeysInt + 1
	}

	objects, err := bucket.ListObjectsV2(ctx, core.ListObjectsV2Input{
		MaxKeys:           requestMaxKeys,
		Prefix:            prefix,
		Delimiter:         delimiter,
		ContinuationToken: v1ContinuationToken,
	})
	if err != nil {
		return listBucketResult{}, err
	}

	// S3 ListObjects v1 marker is exclusive ("start after"); backend continuation token is inclusive ("start from").
	// When marker was sent, strip the first result if it equals the marker.
	if marker != "" && len(objects.Objects) > 0 && objects.Objects[0].Key() == marker {
		objects = &core.ListV2Result{
			Objects:           objects.Objects[1:],
			CommonPrefixes:    objects.CommonPrefixes,
			ContinuationToken: objects.ContinuationToken,
			IsTruncated:       objects.IsTruncated,
		}
	}

	var nextMarker *string

	if delimiter != "" && objects.IsTruncated && objects.ContinuationToken != nil {
		if decoded, err := base64.StdEncoding.DecodeString(*objects.ContinuationToken); err == nil {
			nextMarker = lo.ToPtr(string(decoded))
		}
	}

	return listBucketResult{
		Contents:       mapObjectsToTypes(objects.Objects),
		IsTruncated:    objects.IsTruncated,
		Marker:         marker,
		NextMarker:     nextMarker,
		Name:           bucket.Name(),
		Prefix:         prefix,
		Delimiter:      delimiter,
		MaxKeys:        maxKeysInt,
		CommonPrefixes: lo.Map(objects.CommonPrefixes, func(p string, _ int) prefixEntry { return prefixEntry{Prefix: p} }),
	}, nil
}

func (a APIObjects) ListObjectsV2(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	prefix := c.QueryParam("prefix")
	delimiter := c.QueryParam("delimiter")
	listType := c.QueryParam("list-type")
	maxKeys := c.QueryParam("max-keys")
	continuationToken := c.QueryParam("continuation-token")
	marker := c.QueryParam("marker")

	maxKeysInt, err := validateMaxParam(maxKeys, core.MaxKeys)
	if err != nil {
		return err
	}

	if listType == "2" {
		objects, err := bucket.ListObjectsV2(c.Request().Context(), core.ListObjectsV2Input{
			MaxKeys:           maxKeysInt,
			Prefix:            prefix,
			Delimiter:         delimiter,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return err
		}

		return c.XML(http.StatusOK, listObjectsV2Result{
			Contents:              mapObjectsToTypes(objects.Objects),
			Name:                  bucket.Name(),
			Prefix:                prefix,
			Delimiter:             delimiter,
			MaxKeys:               maxKeysInt,
			KeyCount:              len(objects.Objects) + len(objects.CommonPrefixes),
			NextContinuationToken: objects.ContinuationToken,
			IsTruncated:           objects.IsTruncated,
			CommonPrefixes: lo.Map(objects.CommonPrefixes, func(p string, _ int) prefixEntry {
				return prefixEntry{Prefix: p}
			}),
		})
	}

	resp, err := listObjectsV1Response(
		c.Request().Context(), bucket, prefix, delimiter, marker, maxKeysInt)
	if err != nil {
		return err
	}

	return c.XML(http.StatusOK, resp)
}

func (a APIObjects) ListMultipartUploads(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	prefix := c.QueryParam("prefix")
	delimiter := c.QueryParam("delimiter")
	maxUploadsParam := c.QueryParam("max-uploads")
	keyMarker := c.QueryParam("key-marker")
	uploadIDMarker := c.QueryParam("upload-id-marker")

	maxUploads, err := validateMaxParam(maxUploadsParam, core.MaxUploads)
	if err != nil {
		return err
	}

	result, err := bucket.ListMultipartUploads(c.Request().Context(), core.ListMultipartUploadsInput{
		Prefix:         prefix,
		Delimiter:      delimiter,
		MaxUploads:     maxUploads,
		KeyMarker:      keyMarker,
		UploadIDMarker: uploadIDMarker,
	})
	if err != nil {
		return err
	}

	xmlResponse := listMultipartUploadsResultXML{
		Bucket:         bucket.Name(),
		KeyMarker:      keyMarker,
		UploadIDMarker: uploadIDMarker,
		Prefix:         result.Prefix,
		Delimiter:      result.Delimiter,
		MaxUploads:     result.MaxUploads,
		IsTruncated:    result.IsTruncated,
		Uploads: lo.Map(result.Uploads, func(u core.MultipartUploadInfo, _ int) listMultipartUploadEntryXML {
			return listMultipartUploadEntryXML{
				Key:       u.Key,
				UploadID:  u.UploadID,
				Initiated: u.Initiated.UTC().Format("2006-01-02T15:04:05.000Z"),
			}
		}),
		CommonPrefixes: lo.Map(result.CommonPrefixes, func(p string, _ int) prefixEntry {
			return prefixEntry{Prefix: p}
		}),
	}
	if result.NextKeyMarker != nil {
		xmlResponse.NextKeyMarker = *result.NextKeyMarker
	}

	if result.NextUploadIDMarker != nil {
		xmlResponse.NextUploadIDMarker = *result.NextUploadIDMarker
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
	if err := xml.NewDecoder(io.LimitReader(c.Request().Body, core.SizeLimit1Mb)).Decode(&deleteReq); err != nil {
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

	for _, key := range keys {
		if err := core.ValidateObjectKey(key); err != nil {
			return err
		}
	}

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

	if err := ValidateTags(tags); err != nil {
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
		return core.ErrInvalidPartNumber
	}

	if err := core.ValidatePartNumber(partNumberInt); err != nil {
		return err
	}

	etag, err := bucket.UploadPart(c.Request().Context(), key, uploadID, partNumberInt, c.Request().Body)
	if err != nil {
		return err
	}

	c.Response().Header().Set("ETag", etag)

	return c.NoContent(http.StatusOK)
}

func (a APIObjects) CompleteMultipartUpload(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	key := c.Param("*")
	uploadID := c.QueryParam("uploadId")

	var req completeMultipartUploadRequestXML
	if err := xml.NewDecoder(io.LimitReader(c.Request().Body, core.SizeLimit1Mb)).Decode(&req); err != nil {
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

	for _, part := range parts {
		if err := core.ValidatePartNumber(part.PartNumber); err != nil {
			return err
		}

		if strings.Trim(part.ETag, "\"") == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid part ETag")
		}
	}

	metadata, err := bucket.CompleteMultipartUpload(c.Request().Context(), key, uploadID, parts)
	if err != nil {
		return err
	}

	response := completeMultipartUploadResultXML{
		Bucket: bucket.Name(),
		Key:    key,
	}

	if metadata != nil {
		response.ETag = metadata.SHA256
		c.Response().Header().Set("ETag", metadata.SHA256)
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

func (a APIObjects) ListParts(c *echo.Context) error {
	bucket := apictx.FromContext(c.Request().Context()).Bucket
	apiCtx := apictx.FromContext(c.Request().Context())
	key := c.Param("*")
	uploadID := c.QueryParam("uploadId")

	maxParts, partNumberMarker, err := parseListPartsParams(c)
	if err != nil {
		return err
	}

	result, err := bucket.ListParts(c.Request().Context(), key, core.ListPartsInput{
		UploadID:         uploadID,
		MaxParts:         maxParts,
		PartNumberMarker: partNumberMarker,
	})
	if err != nil {
		return err
	}

	response := listPartsResultFromCore(bucket.Name(), key, uploadID, apiCtx.User, result)

	return c.XML(http.StatusOK, response)
}

// validateMaxParam parses a query param as an integer and ensures it is in [1, maxVal].
// If param is empty, returns maxVal. Otherwise returns error when parse fails or value is out of range.
func validateMaxParam(param string, maxVal int) (int, error) {
	if param == "" {
		return maxVal, nil
	}

	n, err := strconv.Atoi(param)
	if err != nil || n < 1 || n > maxVal {
		return 0, fmt.Errorf("%w: %s must be between 1 and %d", core.ErrInvalidLimitParam, param, maxVal)
	}

	return n, nil
}

func parseListPartsParams(c *echo.Context) (int, int, error) {
	maxPartsParam := c.QueryParam("max-parts")
	partNumberMarkerParam := c.QueryParam("part-number-marker")

	maxParts, err := validateMaxParam(maxPartsParam, core.MaxParts)
	if err != nil {
		return 0, 0, err
	}

	partNumberMarker := 0

	if partNumberMarkerParam != "" {
		var err error

		partNumberMarker, err = strconv.Atoi(partNumberMarkerParam)
		if err != nil {
			return 0, 0, core.ErrInvalidPartNumber
		}

		if partNumberMarker > 0 && core.ValidatePartNumber(partNumberMarker) != nil {
			return 0, 0, core.ErrInvalidPartNumber
		}
	}

	return maxParts, partNumberMarker, nil
}

func listPartsResultFromCore(
	bucketName, key, uploadID string,
	user *core.User,
	result *core.ListPartsResult,
) listPartsResultXML {
	partsXML := lo.Map(result.Parts, func(p core.PartInfo, _ int) listPartXML {
		return listPartXML{
			PartNumber:   p.PartNumber,
			LastModified: p.LastModified.UTC().Format("2006-01-02T15:04:05.000Z"),
			ETag:         "\"" + p.ETag + "\"",
			Size:         p.Size,
		}
	})

	response := listPartsResultXML{
		Bucket:               bucketName,
		Key:                  key,
		UploadID:             uploadID,
		PartNumberMarker:     result.PartNumberMarker,
		NextPartNumberMarker: result.NextPartNumberMarker,
		MaxParts:             result.MaxParts,
		IsTruncated:          result.IsTruncated,
		Parts:                partsXML,
		StorageClass:         "STANDARD",
	}
	if user != nil {
		response.Owner = &types.Owner{
			DisplayName: &user.Name,
			ID:          &user.AccessKeyID,
		}
		response.Initiator = &types.Initiator{
			DisplayName: &user.Name,
			ID:          &user.AccessKeyID,
		}
	}

	return response
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

func ValidateTags(tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	if len(tags) > maxObjectTagCount {
		return fmt.Errorf("%w: maximum %d tags allowed", core.ErrInvalidTag, maxObjectTagCount)
	}

	for k, v := range tags {
		if utf8.RuneCountInString(k) > maxObjectTagKeyLength {
			return fmt.Errorf("%w: tag key exceeds %d characters", core.ErrInvalidTag, maxObjectTagKeyLength)
		}

		if utf8.RuneCountInString(v) > maxObjectTagValLength {
			return fmt.Errorf("%w: tag value exceeds %d characters", core.ErrInvalidTag, maxObjectTagValLength)
		}
	}

	return nil
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
