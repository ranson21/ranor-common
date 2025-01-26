package cors

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int
}

func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         86400,
	}
}

func CORS(config *CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("Access-Control-Allow-Methods", joinStrings(config.AllowedMethods))
			c.Writer.Header().Set("Access-Control-Allow-Headers", joinStrings(config.AllowedHeaders))
			c.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
func joinStrings(strings []string) string {
	if len(strings) == 0 {
		return ""
	}
	result := strings[0]
	for _, s := range strings[1:] {
		result += ", " + s
	}
	return result
}
