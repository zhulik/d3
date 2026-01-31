package middlewares

import (
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/zhulik/d3/internal/apictx"
)

func Logger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:       true,
		LogRemoteIP:      true,
		LogHost:          true,
		LogMethod:        true,
		LogURI:           true,
		LogRequestID:     true,
		LogUserAgent:     true,
		LogStatus:        true,
		LogContentLength: true,
		LogResponseSize:  true,
		HandleError:      true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			logger := c.Logger()
			apiCtx := apictx.FromContext(c.Request().Context())
			commonAttrs := []slog.Attr{
				slog.String("method", apiCtx.Method),
				slog.String("uri", apiCtx.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
				slog.String("host", apiCtx.Host),
				slog.Int64("bytes_in", apiCtx.ContentLength),
				slog.Int64("bytes_out", v.ResponseSize),
				slog.String("user_agent", apiCtx.UserAgent),
				slog.String("remote_ip", apiCtx.RemoteAddr),
				slog.String("request_id", apiCtx.RequestID),
				slog.String("action", string(apiCtx.Action)),
			}
			if v.Error != nil {
				commonAttrs = append(commonAttrs, slog.String("error", v.Error.Error()))
			}

			logger.LogAttrs(c.Request().Context(), slog.LevelInfo, "REQUEST", commonAttrs...)
			return nil
		},
	})
}
