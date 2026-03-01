package middlewares

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/sigv4"
)

func isSigV4AuthError(err error) bool {
	return errors.Is(err, sigv4.ErrSignatureDoesNotMatch) ||
		errors.Is(err, sigv4.ErrInvalidAccessKeyID) ||
		errors.Is(err, sigv4.ErrInvalidDigest) ||
		errors.Is(err, sigv4.ErrMissingDateHeader) ||
		errors.Is(err, sigv4.ErrExpiredPresignRequest) ||
		errors.Is(err, sigv4.ErrMalformedPresignedDate) ||
		errors.Is(err, sigv4.ErrCredMalformed) ||
		errors.Is(err, sigv4.ErrRequestNotReadyYet)
}

func ErrorRenderer() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			switch {
			case isSigV4AuthError(err):
				return echo.NewHTTPError(http.StatusForbidden, err.Error())
			case errors.Is(err, core.ErrBucketNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, core.ErrObjectNotFound) ||
				errors.Is(err, core.ErrPolicyNotFound) ||
				errors.Is(err, core.ErrUserNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, core.ErrBucketAlreadyExists) ||
				errors.Is(err, core.ErrObjectAlreadyExists) ||
				errors.Is(err, core.ErrPolicyAlreadyExists) ||
				errors.Is(err, core.ErrUserAlreadyExists):
				return echo.NewHTTPError(http.StatusConflict, err.Error())
			case errors.Is(err, core.ErrBucketNotEmpty):
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			case errors.Is(err, core.ErrInvalidBucketName) ||
				errors.Is(err, core.ErrInvalidObjectKey) ||
				errors.Is(err, core.ErrInvalidUploadID) ||
				errors.Is(err, core.ErrInvalidMaxKeys) ||
				errors.Is(err, core.ErrPathTraversal) ||
				errors.Is(err, core.ErrSymlinkNotAllowed):
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			case errors.Is(err, core.ErrUnauthorized):
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			case errors.Is(err, iampol.ErrInvalidPolicy):
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			case err == nil:
				return nil
			default:
				return err
			}
		}
	}
}
