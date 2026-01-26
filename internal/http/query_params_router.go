package http //nolint:revive

import (
	"net/http"

	"github.com/labstack/echo/v5"
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

func (r *QueryParamsRouter) AddRoute(param string, handler echo.HandlerFunc) *QueryParamsRouter {
	r.routes = append(r.routes, route{param: param, handler: handler})
	return r
}

func (r *QueryParamsRouter) SetFallbackHandler(handler echo.HandlerFunc) *QueryParamsRouter {
	if r.fallbackHandler != nil {
		panic("fallback handler already set")
	}
	r.fallbackHandler = handler
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
