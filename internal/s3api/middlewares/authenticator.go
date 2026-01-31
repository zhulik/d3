package middlewares

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/sigv4"
)

type credentialStore struct {
	core.UserRepository
}

func (c credentialStore) Get(ctx context.Context, accessKey string) (string, error) {
	user, err := c.GetUserByAccessKeyID(ctx, accessKey)
	if err != nil {
		return "", err
	}
	return user.SecretAccessKey, nil
}

type Authenticator struct {
	UserRepository core.UserRepository
	Logger         *slog.Logger

	credentialStore sigv4.CredentialStore
}

func (a *Authenticator) Init(_ context.Context) error {
	a.credentialStore = credentialStore{a.UserRepository}
	return nil
}

func (a *Authenticator) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			accessKey, err := sigv4.Validate(c.Request().Context(), c.Request(), a.credentialStore)
			if err != nil {
				a.Logger.Error("failed to validate credentials", "error", err)

				// We allow anonymous access to the API, the authorization mechanism will handle it
				return next(c)
			}

			user, err := a.UserRepository.GetUserByAccessKeyID(c.Request().Context(), accessKey)
			if err != nil {
				return err
			}

			apiCtx := apictx.FromContext(c.Request().Context())
			apiCtx.Username = &user.Name

			return next(c)
		}
	}
}
