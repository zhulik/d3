package middlewares

import (
	"context"
	"errors"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/sigv4"
)

type Authenticator struct {
	ManagementBackend core.ManagementBackend
	Logger            *slog.Logger
}

func (a *Authenticator) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authParams, err := sigv4.Validate(c.Request().Context(), c.Request(), a.getAccessKeySecret)
			if err != nil {
				if errors.Is(err, sigv4.ErrRequestNotSigned) {
					// Allow anonymous access, actual authorization is handled by the authorizer
					return next(c)
				}

				a.Logger.Error("failed to validate credentials", "error", err)

				return err
			}

			if authParams != nil {
				user, err := a.ManagementBackend.GetUserByAccessKeyID(c.Request().Context(), authParams.AccessKey)
				if err != nil {
					return err
				}

				apiCtx := apictx.FromContext(c.Request().Context())
				apiCtx.User = user
				apiCtx.AuthParams = authParams
			}

			return next(c)
		}
	}
}

func (a *Authenticator) getAccessKeySecret(ctx context.Context, accessKey string) (string, error) {
	user, err := a.ManagementBackend.GetUserByAccessKeyID(ctx, accessKey)
	if err != nil {
		return "", err
	}

	return user.SecretAccessKey, nil
}
