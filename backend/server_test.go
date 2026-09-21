package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServeStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server := &http.Server{Addr: "127.0.0.1:0"}
	if err := serve(ctx, server, time.Second); err != nil {
		t.Fatalf("serve() error = %v", err)
	}
}

func TestServeReturnsStartupError(t *testing.T) {
	server := &http.Server{Addr: "invalid-address"}
	err := serve(context.Background(), server, time.Second)
	if err == nil {
		t.Fatal("serve() error = nil, want startup error")
	}
	if !strings.Contains(err.Error(), "HTTPサーバーの起動に失敗しました") {
		t.Fatalf("serve() error = %q, want startup error", err)
	}
}
