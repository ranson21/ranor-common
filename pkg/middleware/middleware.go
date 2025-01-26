package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ranson21/ranor-common/pkg/logger"
	"github.com/ranson21/ranor-common/pkg/middleware/context"
	"github.com/ranson21/ranor-common/pkg/middleware/cors"
	logMiddleware "github.com/ranson21/ranor-common/pkg/middleware/logger"
	"github.com/ranson21/ranor-common/pkg/middleware/ratelimit"
	"github.com/ranson21/ranor-common/pkg/middleware/recovery"
)

func DefaultMiddlewares(log logger.Logger) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		context.GinContextToContextMiddleware(),
		cors.CORS(cors.DefaultCORSConfig()),
		logMiddleware.Logger(log),
		recovery.Recovery(log),
		ratelimit.NewRateLimiter(10, 20).RateLimit(),
	}
}
