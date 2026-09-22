package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func serve(ctx context.Context, server *http.Server, timeout time.Duration) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("HTTPサーバーの起動に失敗しました: %w", err)
	case <-ctx.Done():
		log.Print("HTTPサーバーの停止を開始します")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("HTTPサーバーの停止に失敗しました: %w", err)
	}

	if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTPサーバーの停止中にエラーが発生しました: %w", err)
	}

	log.Print("HTTPサーバーを正常に停止しました")
	return nil
}
