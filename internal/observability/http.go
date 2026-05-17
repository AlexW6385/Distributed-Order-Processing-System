package observability

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func HTTPMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()

		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = NewRequestID()
		}

		c.Header(RequestIDHeader, requestID)
		c.Request = c.Request.WithContext(ContextWithRequestID(c.Request.Context(), requestID))

		c.Next()

		slog.InfoContext(
			c.Request.Context(),
			"http request completed",
			slog.String("service", serviceName),
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.FullPath()),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(startedAt)),
		)
	}
}
