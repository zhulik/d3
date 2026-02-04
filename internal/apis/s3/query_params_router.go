package s3

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/samber/lo/mutable"
	"github.com/zhulik/d3/internal/apis/s3/actions"
	"github.com/zhulik/d3/internal/apis/s3/middlewares"
)

type route struct {
	param   string
	handler echo.HandlerFunc
}

type QueryParamsRouter struct {
	routes []route

	fallbackHandler echo.HandlerFunc
}

func NewQueryParamsRouter() *QueryParamsRouter {
	return &QueryParamsRouter{
		routes: []route{},
	}
}

func (r *QueryParamsRouter) AddRoute(param string, handler echo.HandlerFunc, action actions.Action,
	moreMiddlewares ...echo.MiddlewareFunc) *QueryParamsRouter {
	allMiddlewares := []echo.MiddlewareFunc{middlewares.SetAction(action)}
	allMiddlewares = append(allMiddlewares, moreMiddlewares...)

	r.routes = append(r.routes, route{param: param, handler: applyMiddlewares(handler, allMiddlewares...)})

	return r
}

func (r *QueryParamsRouter) SetFallbackHandler(handler echo.HandlerFunc, action actions.Action,
	moreMiddlewares ...echo.MiddlewareFunc) *QueryParamsRouter {
	if r.fallbackHandler != nil {
		panic("fallback handler already set")
	}

	allMiddlewares := []echo.MiddlewareFunc{middlewares.SetAction(action)}
	allMiddlewares = append(allMiddlewares, moreMiddlewares...)

	r.fallbackHandler = applyMiddlewares(handler, allMiddlewares...)

	return r
}

func (r *QueryParamsRouter) Handle(c *echo.Context) error {
	for _, route := range r.routes {
		if _, ok := c.QueryParams()[route.param]; ok {
			return route.handler(c)
		}
	}

	if r.fallbackHandler != nil {
		return r.fallbackHandler(c)
	}

	return echo.NewHTTPError(http.StatusNotImplemented, "not implemented")
}

func applyMiddlewares(h echo.HandlerFunc, middlewares ...echo.MiddlewareFunc) echo.HandlerFunc {
	mutable.Reverse(middlewares)

	for _, middleware := range middlewares {
		h = middleware(h)
	}

	return h
}
