package middlewares

import (
	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
)

func BucketNameValidator(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if err := core.ValidateBucketName(c.Param("bucket")); err != nil {
			return err
		}

		return next(c)
	}
}

func ObjectKeyValidator(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if err := core.ValidateObjectKey(c.Param("*")); err != nil {
			return err
		}

		return next(c)
	}
}

func UploadIDValidator(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if err := core.ValidateUploadID(c.QueryParam("uploadId")); err != nil {
			return err
		}

		return next(c)
	}
}
