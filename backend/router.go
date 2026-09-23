package main

import (
	"io"
	"log"
	"net/http"

	"backend/api"
	"backend/handler"
	"backend/middleware"

	"github.com/gin-gonic/gin"
)

type authService interface {
	handler.AuthService
	middleware.AuthService
}

// アクセスログ、panic復旧、CORS、Origin検証を全ルートへ一貫して適用する。
func newRouter(allowedOrigins []string, authService authService, authHandler *handler.AuthHandler, isRender bool) *gin.Engine {
	router := gin.New()
	configureClientIP(router, isRender)
	router.Use(
		gin.LoggerWithConfig(gin.LoggerConfig{SkipQueryString: true}),
		gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, _ any) {
			log.Print("HTTPリクエスト処理中のpanicをRecoveryしました")
			api.InternalServerError(c)
		}),
		middleware.CORS(allowedOrigins),
		middleware.RequireAllowedOrigin(allowedOrigins),
	)

	router.GET("/health", healthHandler)
	auth := router.Group("/api/auth")
	auth.Use(middleware.NoStore())
	auth.POST("/login", middleware.LoginRateLimit(), authHandler.Login)
	auth.GET("/me", middleware.RequireAuthentication(authService), authHandler.Me)
	auth.POST("/logout", authHandler.Logout)
	router.NoRoute(func(c *gin.Context) {
		api.NotFound(c)
	})

	return router
}

// Renderでは外部からコンテナへ直接到達できず、CF-Connecting-IPは入口で上書きされる。
// それ以外の環境では転送ヘッダーを信頼せず、RemoteAddrをクライアントIPとして扱う。
func configureClientIP(router *gin.Engine, isRender bool) {
	trustedProxies := []string(nil)
	router.RemoteIPHeaders = nil
	if isRender {
		trustedProxies = []string{"0.0.0.0/0", "::/0"}
		router.RemoteIPHeaders = []string{"CF-Connecting-IP"}
	}
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		panic("trusted proxy設定に失敗しました")
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
