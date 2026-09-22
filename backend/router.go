package main

import (
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Error string `json:"error"`
}

// アクセスログ、panic復旧、CORSを全ルートへ一貫して適用する。
func newRouter(allowedOrigins []string) *gin.Engine {
	router := gin.New()
	router.Use(
		gin.LoggerWithConfig(gin.LoggerConfig{SkipQueryString: true}),
		gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, _ any) {
			log.Print("HTTPリクエスト処理中のpanicをRecoveryしました")
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				errorResponse{Error: "internal_server_error"},
			)
		}),
		corsMiddleware(allowedOrigins),
	)

	router.GET("/health", healthHandler)
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, errorResponse{Error: "not_found"})
	})

	return router
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// 許可したブラウザーOriginだけにレスポンスの読み取りを認める。
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		c.Writer.Header().Add("Vary", "Origin")

		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if _, ok := allowed[origin]; !ok {
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatusJSON(
					http.StatusForbidden,
					errorResponse{Error: "origin_not_allowed"},
				)
				return
			}

			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)

		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
