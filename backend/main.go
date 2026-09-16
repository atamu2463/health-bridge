package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"backend/config"
)

const serverAddress = ":8080"

// /health はHTTPプロセスの稼働確認に限定し、DB接続はサーバー起動前に検証する。
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprintln(w, "OK"); err != nil {
		log.Printf("ヘルスチェックのレスポンス送信に失敗しました: %v", err)
	}
}

func newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)

	return mux
}

func main() {
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatal("DB接続に失敗しました", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("DBインスタンスの取得に失敗しました", err)
	}

	fmt.Println("DB接続に成功しました")

	defer sqlDB.Close()

	// ヘッダー送信が完了しない接続による、サーバー資源の占有を防止する。
	server := &http.Server{
		Addr:              serverAddress,
		Handler:           newRouter(),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("HTTPサーバーを起動しました: http://localhost%s", serverAddress)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("HTTPサーバーの起動に失敗しました: %v", err)
	}
}
