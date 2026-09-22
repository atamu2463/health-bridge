package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/config"

	"github.com/gin-gonic/gin"
)

const (
	readTimeout       = 5 * time.Second
	readHeaderTimeout = 5 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// OSシグナルと終了コードだけを扱い、初期化から停止までの処理はrunServerへ集約する。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runServer(ctx); err != nil {
		log.Printf("バックエンドを終了しました: %v", err)
		os.Exit(1)
	}
}

func runServer(ctx context.Context) (runErr error) {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := config.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("DB接続に失敗しました: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("DBインスタンスの取得に失敗しました: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil && runErr == nil {
			runErr = fmt.Errorf("DB接続の終了に失敗しました: %w", err)
		}
	}()

	gin.SetMode(gin.ReleaseMode)

	// ヘッダー送信が完了しない接続によるサーバー資源の占有を防止する。
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           newRouter(cfg.AllowedOrigins),
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	log.Printf("HTTPサーバーを起動しました: port=%s", cfg.Port)

	return serve(ctx, server, shutdownTimeout)
}
