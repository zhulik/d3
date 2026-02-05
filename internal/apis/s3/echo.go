package s3

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/zhulik/d3/internal/apictx"
	middlewares2 "github.com/zhulik/d3/internal/apis/s3/middlewares"
	"github.com/zhulik/d3/pkg/s3actions"
)

type Echo struct {
	*echo.Echo

	Auth *middlewares2.Authenticator

	rootQueryRouter *QueryParamsRouter
}

func (e *Echo) Init(_ context.Context) error {
	e.Echo = echo.New()
	e.Logger = slog.Default()
	e.rootQueryRouter = NewQueryParamsRouter()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(
		apictx.Middleware(),
		middlewares2.Logger(),
		middleware.Recover(),
		apictx.Middleware(),
		middlewares2.ErrorRenderer(),
		e.Auth.Middleware(),
	)

	return nil
}

func (e *Echo) AddQueryParamRoute(path string, handler echo.HandlerFunc, action s3actions.Action, middlewares ...echo.MiddlewareFunc) { //nolint:lll
	e.rootQueryRouter.AddRoute(path, applyMiddlewares(handler, middlewares...), action)
}
