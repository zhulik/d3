package s3api

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/s3api/actions"
	"github.com/zhulik/d3/internal/s3api/middlewares"
)

type Echo struct {
	*echo.Echo

	Auth *middlewares.Authenticator

	rootQueryRouter *QueryParamsRouter
}

func (e *Echo) Init(_ context.Context) error {
	e.Echo = echo.New()
	e.Logger = slog.Default()
	e.rootQueryRouter = NewQueryParamsRouter()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(
		apictx.Middleware(),
		middlewares.Logger(),
		middleware.Recover(),
		apictx.Middleware(),
		middlewares.ErrorRenderer(),
		e.Auth.Middleware(),
	)

	return nil
}

func (e *Echo) AddQueryParamRoute(path string, handler echo.HandlerFunc, action actions.Action, middlewares ...echo.MiddlewareFunc) { //nolint:lll
	e.rootQueryRouter.AddRoute(path, applyMiddlewares(handler, middlewares...), action)
}
