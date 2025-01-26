package recovery

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/ranson21/ranor-common/pkg/logger"
	"go.uber.org/zap"
)

func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("url", c.Request.URL.String()),
					zap.String("method", c.Request.Method),
				)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				c.Abort()
			}
		}()
		c.Next()
	}
}
