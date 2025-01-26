package context

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

// GinContextFromContext --  Map the Gin context to the go context
func GinContextFromContext(ctx context.Context) *gin.Context {
	ginContext := ctx.Value("GinContextKey")
	if ginContext == nil {
		log.Printf("Something went wrong: %v", fmt.Errorf("could not retrieve gin.Context"))
		return nil
	}

	gc, ok := ginContext.(*gin.Context)
	if !ok {
		log.Printf("Something went wrong: %v", fmt.Errorf("gin.Context has wrong type"))
		return nil
	}
	return gc
}

// GinContextToContextMiddleware -- Set the gin context on the request context
func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
