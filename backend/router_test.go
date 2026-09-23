package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"backend/handler"
	"backend/service"

	"github.com/gin-gonic/gin"
)

const testAllowedOrigin = "http://localhost:3000"

type routerAuthService struct{}

func (routerAuthService) Login(context.Context, string, string) (service.LoginResult, error) {
	return service.LoginResult{}, service.ErrInvalidCredentials
}

func (routerAuthService) Logout(context.Context, string) error {
	return nil
}

func (routerAuthService) Authenticate(context.Context, string) (service.AuthenticatedUser, error) {
	return service.AuthenticatedUser{}, service.ErrUnauthenticated
}

func newTestRouter() *gin.Engine {
	authService := routerAuthService{}
	return newRouter([]string{testAllowedOrigin}, authService, handler.NewAuthHandler(authService, false), false)
}

func TestHealthEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	newTestRouter().ServeHTTP(response, request)

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

	newTestRouter().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusNotFound)
	}

	if body := response.Body.String(); !strings.Contains(body, `"error":{"code":"not_found","message":"`) {
		t.Fatalf("response body does not use the common error format: %q", body)
	}
}

func TestRecoveryDoesNotExposeInternalError(t *testing.T) {
	router := newTestRouter()
	router.GET("/panic", func(_ *gin.Context) {
		panic("内部エラーの詳細")
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	if body := response.Body.String(); !strings.Contains(body, `"error":{"code":"internal_server_error","message":"`) {
		t.Fatalf("response body does not use the common error format: %q", body)
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

			newTestRouter().ServeHTTP(response, request)

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
			wantCredentials := ""
			if tt.wantAllowOrigin != "" {
				wantCredentials = "true"
			}
			if allowCredentials := response.Header().Get("Access-Control-Allow-Credentials"); allowCredentials != wantCredentials {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want %q", allowCredentials, wantCredentials)
			}
			if tt.wantAllowOrigin != "" {
				if methods := response.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, PUT, PATCH, DELETE, OPTIONS" {
					t.Fatalf("Access-Control-Allow-Methods = %q", methods)
				}
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

			newTestRouter().ServeHTTP(response, request)

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
			wantCredentials := ""
			if tt.wantAllowOrigin != "" {
				wantCredentials = "true"
			}
			if credentials := response.Header().Get("Access-Control-Allow-Credentials"); credentials != wantCredentials {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want %q", credentials, wantCredentials)
			}
		})
	}
}

func TestStateChangingRequestsRequireAllowedOrigin(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		wantStatus int
	}{
		{name: "allowed", origin: testAllowedOrigin, wantStatus: http.StatusBadRequest},
		{name: "missing", wantStatus: http.StatusForbidden},
		{name: "not allowed", origin: "https://example.com", wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{}`))
			request.Header.Set("Content-Type", "application/json")
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			response := httptest.NewRecorder()
			newTestRouter().ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if tt.wantStatus == http.StatusForbidden && !strings.Contains(response.Body.String(), `"code":"origin_not_allowed"`) {
				t.Fatalf("unexpected error body: %s", response.Body.String())
			}
			if tt.origin != testAllowedOrigin && response.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatal("credentials header was returned for a missing or disallowed Origin")
			}
		})
	}
}

func TestAuthEndpointsDisableResponseCaching(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "login", method: http.MethodPost, path: "/api/auth/login", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "me", method: http.MethodGet, path: "/api/auth/me", wantStatus: http.StatusUnauthorized},
		{name: "logout", method: http.MethodPost, path: "/api/auth/logout", wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			if tt.method != http.MethodGet {
				request.Header.Set("Origin", testAllowedOrigin)
			}
			response := httptest.NewRecorder()
			newTestRouter().ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
				t.Fatalf("Cache-Control = %q, want %q", cacheControl, "no-store")
			}
		})
	}
}

func TestLoginRateLimitIsAppliedToRouter(t *testing.T) {
	router := newTestRouter()
	for attempt := 1; attempt <= 31; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", testAllowedOrigin)
		request.RemoteAddr = "192.0.2.10:1234"
		// 転送ヘッダーを書き換えても、非Render環境では送信元IPの制限を回避できない。
		request.Header.Set("X-Forwarded-For", "198.51.100."+strconv.Itoa(attempt))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if attempt <= 30 {
			if response.Code != http.StatusBadRequest {
				t.Fatalf("attempt %d status = %d, want %d", attempt, response.Code, http.StatusBadRequest)
			}
			continue
		}
		if response.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = %d, want %d", attempt, response.Code, http.StatusTooManyRequests)
		}
		if !strings.Contains(response.Body.String(), `"code":"too_many_requests"`) {
			t.Fatalf("unexpected error body: %s", response.Body.String())
		}
		if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
			t.Fatalf("Cache-Control = %q, want %q", cacheControl, "no-store")
		}
	}
}

func TestAccessLogAndErrorsDoNotContainAuthenticationSecrets(t *testing.T) {
	previousWriter := gin.DefaultWriter
	var logOutput bytes.Buffer
	gin.DefaultWriter = &logOutput
	t.Cleanup(func() { gin.DefaultWriter = previousWriter })

	email := "private@example.invalid"
	password := "password-secret"
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"email":"`+email+`","password":"`+password+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", testAllowedOrigin)
	request.Header.Set("Cookie", "health_bridge_session=cookie-secret")
	response := httptest.NewRecorder()
	newTestRouter().ServeHTTP(response, request)

	for _, secret := range []string{email, password, "cookie-secret"} {
		if strings.Contains(logOutput.String(), secret) {
			t.Fatalf("access log contains secret %q", secret)
		}
		if strings.Contains(response.Body.String(), secret) {
			t.Fatalf("error response contains secret %q", secret)
		}
	}
}

func TestClientIPProxyTrust(t *testing.T) {
	tests := []struct {
		name           string
		isRender       bool
		cfConnectingIP string
		wantIP         string
	}{
		{name: "転送ヘッダーを既定では信頼しない", cfConnectingIP: "203.0.113.20", wantIP: "192.0.2.10"},
		{name: "RenderではCF-Connecting-IPを使用する", isRender: true, cfConnectingIP: "203.0.113.20", wantIP: "203.0.113.20"},
		{name: "Renderでヘッダーがない場合は接続元IPへ戻る", isRender: true, wantIP: "192.0.2.10"},
		{name: "Renderでヘッダーが不正な場合は接続元IPへ戻る", isRender: true, cfConnectingIP: "invalid-ip", wantIP: "192.0.2.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			configureClientIP(router, tt.isRender)
			router.GET("/client-ip", func(c *gin.Context) {
				c.String(http.StatusOK, c.ClientIP())
			})

			request := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
			request.RemoteAddr = "192.0.2.10:1234"
			request.Header.Set("X-Forwarded-For", "198.51.100.30")
			if tt.cfConnectingIP != "" {
				request.Header.Set("CF-Connecting-IP", tt.cfConnectingIP)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if body := response.Body.String(); body != tt.wantIP {
				t.Fatalf("ClientIP = %q, want %q", body, tt.wantIP)
			}
		})
	}
}
