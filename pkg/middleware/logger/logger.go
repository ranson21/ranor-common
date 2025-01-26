package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ranson21/ranor-common/pkg/logger"
	"go.uber.org/zap"
)

func Logger(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("Incoming request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("remote_addr", c.ClientIP()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
