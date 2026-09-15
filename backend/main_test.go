package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	newRouter().ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", result.StatusCode, http.StatusOK)
	}

	contentType := result.Header.Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf(
			"Content-Type = %q, want %q",
			contentType,
			"text/plain; charset=utf-8",
		)
	}

	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("レスポンス本文の読み取りに失敗しました: %v", err)
	}

	if string(body) != "OK\n" {
		t.Fatalf("response body = %q, want %q", string(body), "OK\n")
	}
}
