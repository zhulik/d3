package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
)

func (s *Server) HeadObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	result, err := s.Backend.HeadObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Last-Modified", result.LastModified.Format(http.TimeFormat))
	c.Response().Header().Set("Content-Length", strconv.FormatInt(result.ContentLength, 10))

	return c.NoContent(http.StatusOK)
}

func (s *Server) PutObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	err := s.Backend.PutObject(c.Request().Context(), bucket, key, c.Request().Body)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) GetObject(c *echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("*")

	contents, err := s.Backend.GetObject(c.Request().Context(), bucket, key)
	if err != nil {
		return err
	}

	defer contents.Close()

	c.Response().Header().Set("Last-Modified", contents.LastModified.Format(http.TimeFormat))
	c.Response().Header().Set("Content-Length", strconv.FormatInt(contents.Size, 10))
	c.Response().Header().Set("ETag", fmt.Sprintf("%x", "foo"))

	return c.Stream(http.StatusOK, "application/octet-stream", contents)
}
