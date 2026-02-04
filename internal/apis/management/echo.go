package management

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/zhulik/d3/internal/apictx"
	middlewares2 "github.com/zhulik/d3/internal/apis/s3/middlewares"
)

type Echo struct {
	*echo.Echo

	Auth *middlewares2.Authenticator
}

func (e *Echo) Init(_ context.Context) error {
	e.Echo = echo.New()
	e.Logger = slog.Default()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(
		apictx.Middleware(),
		middlewares2.Logger(),
		middleware.Recover(),
		apictx.Middleware(),
		e.Auth.Middleware(),
	)

	return nil
}
