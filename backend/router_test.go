package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

const testAllowedOrigin = "http://localhost:3000"

func TestHealthEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	newRouter([]string{testAllowedOrigin}).ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", result.StatusCode, http.StatusOK)
	}

	if contentType := result.Header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf(
			"Content-Type = %q, want %q",
			contentType,
			"application/json; charset=utf-8",
		)
	}

	if body := response.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("response body = %q, want %q", body, `{"status":"ok"}`)
	}
}

func TestUnknownRoute(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	response := httptest.NewRecorder()

	newRouter([]string{testAllowedOrigin}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusNotFound)
	}

	if body := response.Body.String(); body != `{"error":"not_found"}` {
		t.Fatalf("response body = %q, want %q", body, `{"error":"not_found"}`)
	}
}

func TestRecoveryDoesNotExposeInternalError(t *testing.T) {
	router := newRouter([]string{testAllowedOrigin})
	router.GET("/panic", func(_ *gin.Context) {
		panic("内部エラーの詳細")
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	if body := response.Body.String(); body != `{"error":"internal_server_error"}` {
		t.Fatalf(
			"response body = %q, want %q",
			body,
			`{"error":"internal_server_error"}`,
		)
	}
}

func TestCORSPreflight(t *testing.T) {
	tests := []struct {
		name            string
		origin          string
		wantStatus      int
		wantAllowOrigin string
	}{
		{
			name:            "許可したOrigin",
			origin:          testAllowedOrigin,
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: testAllowedOrigin,
		},
		{
			name:       "許可していないOrigin",
			origin:     "https://example.com",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, "/health", nil)
			request.Header.Set("Origin", tt.origin)
			request.Header.Set("Access-Control-Request-Method", http.MethodGet)
			response := httptest.NewRecorder()

			newRouter([]string{testAllowedOrigin}).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status code = %d, want %d", response.Code, tt.wantStatus)
			}
			if allowOrigin := response.Header().Get("Access-Control-Allow-Origin"); allowOrigin != tt.wantAllowOrigin {
				t.Fatalf(
					"Access-Control-Allow-Origin = %q, want %q",
					allowOrigin,
					tt.wantAllowOrigin,
				)
			}
			if allowCredentials := response.Header().Get("Access-Control-Allow-Credentials"); allowCredentials != "" {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", allowCredentials)
			}
			if vary := response.Header().Get("Vary"); vary != "Origin" {
				t.Fatalf("Vary = %q, want %q", vary, "Origin")
			}
		})
	}
}

func TestCORSVaryOrigin(t *testing.T) {
	tests := []struct {
		name            string
		origin          string
		wantAllowOrigin string
	}{
		{
			name:            "許可したOrigin",
			origin:          testAllowedOrigin,
			wantAllowOrigin: testAllowedOrigin,
		},
		{
			name:   "許可していないOrigin",
			origin: "https://example.com",
		},
		{
			name: "Originなし",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			response := httptest.NewRecorder()

			newRouter([]string{testAllowedOrigin}).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
			}
			if vary := response.Header().Get("Vary"); vary != "Origin" {
				t.Fatalf("Vary = %q, want %q", vary, "Origin")
			}
			if allowOrigin := response.Header().Get("Access-Control-Allow-Origin"); allowOrigin != tt.wantAllowOrigin {
				t.Fatalf(
					"Access-Control-Allow-Origin = %q, want %q",
					allowOrigin,
					tt.wantAllowOrigin,
				)
			}
		})
	}
}
