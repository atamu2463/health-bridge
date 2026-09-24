package middleware

import (
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

const allowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := originSet(allowedOrigins)

	return func(c *gin.Context) {
		c.Writer.Header().Add("Vary", "Origin")

		origin := c.GetHeader("Origin")
		_, isAllowed := allowed[origin]
		if origin != "" && isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method != http.MethodOptions {
			c.Next()
			return
		}

		if origin == "" || !isAllowed {
			api.OriginNotAllowed(c)
			return
		}

		c.Header("Access-Control-Allow-Methods", allowedMethods)
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.AbortWithStatus(http.StatusNoContent)
	}
}

func RequireAllowedOrigin(allowedOrigins []string) gin.HandlerFunc {
	allowed := originSet(allowedOrigins)

	return func(c *gin.Context) {
		if !isStateChangingMethod(c.Request.Method) {
			c.Next()
			return
		}

		if _, ok := allowed[c.GetHeader("Origin")]; !ok {
			api.OriginNotAllowed(c)
			return
		}

		c.Next()
	}
}

func originSet(origins []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return allowed
}

func isStateChangingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
