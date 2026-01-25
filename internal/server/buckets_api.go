package server

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/labstack/echo/v5"
)

// BucketsResult is the XML envelope for ListBuckets responses.
type BucketsResult struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListAllMyBucketsResult"`
	Owner   *types.Owner
	Buckets []*types.Bucket `xml:"Buckets>Bucket"`
}

// LocationConstraintResponse is the XML response for GetBucketLocation.
type LocationConstraintResponse struct {
	XMLName  xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ LocationConstraint"`
	Location string   `xml:",chardata"`
}

// PrefixEntry represents a common prefix in S3 list results.
type PrefixEntry struct {
	Prefix string `xml:"Prefix"`
}

// ListBucketResult is a minimal representation of S3's ListBucket result.
type ListBucketResult struct {
	IsTruncated    bool            `xml:"IsTruncated"`
	Contents       []*types.Object `xml:"Contents"`
	Name           string          `xml:"Name"`
	Prefix         string          `xml:"Prefix"`
	Delimiter      string          `xml:"Delimiter,omitempty"`
	MaxKeys        int             `xml:"MaxKeys"`
	CommonPrefixes []PrefixEntry   `xml:"CommonPrefixes,omitempty"`
}

// ListBuckets enumerates existing JetStream Object Store buckets and returns
// a simple S3-compatible XML response.
func (s *Server) ListBuckets(c *echo.Context) error {
	entries, err := s.Backend.ListBuckets(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	response := BucketsResult{
		Buckets: entries,
	}

	return c.XML(http.StatusOK, response)
}

func (s *Server) CreateBucket(c *echo.Context) error {
	name := c.Param("bucket")
	err := s.Backend.CreateBucket(c.Request().Context(), name)
	if err != nil {
		return c.String(http.StatusConflict, err.Error())
	}

	c.Response().Header().Set("Location", fmt.Sprintf("/%s", name))
	c.Response().Header().Set("x-amz-bucket-arn", fmt.Sprintf("arn:aws:s3:::%s", name))

	return c.NoContent(http.StatusCreated)
}

func (s *Server) DeleteBucket(c *echo.Context) error {
	name := c.Param("bucket")
	err := s.Backend.DeleteBucket(c.Request().Context(), name)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) GetBucketLocation(c *echo.Context) error {
	bucket := c.Param("bucket")

	// depending on the query params this endpoint may perform different actions
	if _, err := echo.QueryParam[string](c, "location"); err == nil {

		err := s.Backend.HeadBucket(c.Request().Context(), bucket)
		if err != nil {
			return err
		}

		response := LocationConstraintResponse{
			Location: "local",
		}

		return c.XML(http.StatusOK, response)
	}
	if _, err := echo.QueryParam[string](c, "prefix"); err == nil {
		prefix := c.QueryParam("prefix")
		objects, err := s.Backend.ListObjects(c.Request().Context(), bucket, prefix)
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
	panic("unknown query param")
}

func (s *Server) HeadBucket(c *echo.Context) error {
	err := s.Backend.HeadBucket(c.Request().Context(), c.Param("bucket"))
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}
