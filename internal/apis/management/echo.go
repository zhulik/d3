package management

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/zhulik/d3/internal/apictx"
	managementMiddleares "github.com/zhulik/d3/internal/apis/management/middlewares"
	"github.com/zhulik/d3/internal/apis/s3/middlewares"
)

type Echo struct {
	*echo.Echo

	Authenticator *middlewares.Authenticator
	Authorizer    *managementMiddleares.Authorizer
}

func (e *Echo) Init(_ context.Context) error {
	e.Echo = echo.New()
	e.Logger = slog.Default()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(
		apictx.Middleware(),
		middlewares.Logger(),
		middleware.Recover(),
		middlewares.ErrorRenderer(),
		apictx.Middleware(),
		e.Authenticator.Middleware(),
		e.Authorizer.Middleware(),
	)

	return nil
}
