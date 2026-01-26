package http //nolint:revive

import "github.com/labstack/echo/v5"

func SetHeaders(c *echo.Context, headers map[string]string) {
	for key, value := range headers {
		c.Response().Header().Set(key, value)
	}
}
