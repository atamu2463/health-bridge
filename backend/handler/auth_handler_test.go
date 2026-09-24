package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/api"
	"backend/service"

	"github.com/gin-gonic/gin"
)

var handlerTestExpiresAt = time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)

type fakeAuthService struct {
	loginResult service.LoginResult
	loginErr    error
	logoutErr   error
	loginEmail  string
	loginPass   string
	logoutToken string
	logoutCalls int
}

func (s *fakeAuthService) Login(_ context.Context, email, password string) (service.LoginResult, error) {
	s.loginEmail = email
	s.loginPass = password
	return s.loginResult, s.loginErr
}

func (s *fakeAuthService) Logout(_ context.Context, token string) error {
	s.logoutCalls++
	s.logoutToken = token
	return s.logoutErr
}

func TestLoginReturnsUserAndSecureCookie(t *testing.T) {
	managerID := uint(9)
	fake := &fakeAuthService{loginResult: service.LoginResult{
		User: service.AuthenticatedUser{
			ID: 12, Name: "テスト従業員", Email: "employee@example.invalid",
			Role: "employee", ManagerID: &managerID,
		},
		Token: "session-token-secret", ExpiresAt: handlerTestExpiresAt,
	}}
	recorder := performHandlerRequest(http.MethodPost, "/api/auth/login", `{"email":" Employee@Example.Invalid ","password":"password"}`, NewAuthHandler(fake, true).Login, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	wantBody := `{"user":{"id":"12","name":"テスト従業員","email":"employee@example.invalid","role":"employee","managerId":"9"}}`
	if body := recorder.Body.String(); body != wantBody {
		t.Fatalf("body = %q, want %q", body, wantBody)
	}
	if fake.loginEmail != "employee@example.invalid" || fake.loginPass != "password" {
		t.Fatal("Login() did not receive the expected credentials")
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != SessionCookieName || cookie.Value != "session-token-secret" || cookie.Path != "/api" {
		t.Fatalf("unexpected session cookie: %#v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteNoneMode {
		t.Fatalf("secure cookie attributes are incorrect: %#v", cookie)
	}
	if !cookie.Expires.Equal(handlerTestExpiresAt) || cookie.Domain != "" {
		t.Fatalf("cookie expiration or domain is incorrect: %#v", cookie)
	}
	if strings.Contains(recorder.Body.String(), fake.loginResult.Token) {
		t.Fatal("response contains the session token")
	}
}

func TestLoginUsesLocalCookieAttributes(t *testing.T) {
	fake := &fakeAuthService{loginResult: service.LoginResult{
		User:      service.AuthenticatedUser{ID: 1, Role: "manager"},
		Token:     "local-token",
		ExpiresAt: handlerTestExpiresAt,
	}}
	recorder := performHandlerRequest(http.MethodPost, "/api/auth/login", `{"email":"manager@example.invalid","password":"password"}`, NewAuthHandler(fake, false).Login, nil)

	cookie := recorder.Result().Cookies()[0]
	if cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("local cookie attributes are incorrect: %#v", cookie)
	}
}

func TestLoginRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `{"email":`},
		{name: "empty email", body: `{"email":" ","password":"password"}`},
		{name: "empty password", body: `{"email":"user@example.invalid","password":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAuthService{}
			recorder := performHandlerRequest(http.MethodPost, "/api/auth/login", tt.body, NewAuthHandler(fake, false).Login, nil)
			assertErrorResponse(t, recorder, http.StatusBadRequest, api.ErrorCodeInvalidRequest)
			if fake.loginEmail != "" {
				t.Fatal("Login() was called for an invalid request")
			}
		})
	}
}

func TestLoginMapsAuthenticationErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid credentials", err: service.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantCode: api.ErrorCodeInvalidCredentials},
		{name: "internal error", err: service.ErrInternal, wantStatus: http.StatusInternalServerError, wantCode: api.ErrorCodeInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAuthService{loginErr: tt.err}
			recorder := performHandlerRequest(http.MethodPost, "/api/auth/login", `{"email":"secret@example.invalid","password":"password-secret"}`, NewAuthHandler(fake, false).Login, nil)
			assertErrorResponse(t, recorder, tt.wantStatus, tt.wantCode)
			for _, secret := range []string{"secret@example.invalid", "password-secret"} {
				if strings.Contains(recorder.Body.String(), secret) {
					t.Fatalf("response contains secret %q", secret)
				}
			}
		})
	}
}

func TestMeReturnsAuthenticatedUser(t *testing.T) {
	managerID := uint(4)
	user := service.AuthenticatedUser{ID: 7, Name: "従業員", Email: "user@example.invalid", Role: "employee", ManagerID: &managerID}
	recorder := performHandlerRequest(http.MethodGet, "/api/auth/me", "", NewAuthHandler(&fakeAuthService{}, false).Me, &user)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"id":"7"`) || !strings.Contains(recorder.Body.String(), `"managerId":"4"`) {
		t.Fatalf("unexpected me response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestLogoutIsIdempotentAndExpiresCookie(t *testing.T) {
	tests := []struct {
		name      string
		cookie    *http.Cookie
		wantCalls int
		wantToken string
	}{
		{name: "cookie exists", cookie: &http.Cookie{Name: SessionCookieName, Value: "token"}, wantCalls: 1, wantToken: "token"},
		{name: "cookie missing", wantCalls: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAuthService{}
			recorder := performHandlerRequest(http.MethodPost, "/api/auth/logout", "", NewAuthHandler(fake, true).Logout, nil, tt.cookie)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
			}
			if fake.logoutCalls != tt.wantCalls || fake.logoutToken != tt.wantToken {
				t.Fatal("Logout() calls do not match")
			}
			cookie := recorder.Result().Cookies()[0]
			if cookie.Name != SessionCookieName || cookie.Path != "/api" || cookie.MaxAge >= 0 || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteNoneMode {
				t.Fatalf("expired cookie attributes are incorrect: %#v", cookie)
			}
		})
	}
}

func TestLogoutReturnsInternalErrorButStillExpiresCookie(t *testing.T) {
	fake := &fakeAuthService{logoutErr: errors.New("database unavailable")}
	recorder := performHandlerRequest(http.MethodPost, "/api/auth/logout", "", NewAuthHandler(fake, false).Logout, nil, &http.Cookie{Name: SessionCookieName, Value: "token"})
	assertErrorResponse(t, recorder, http.StatusInternalServerError, api.ErrorCodeInternalServerError)
	if len(recorder.Result().Cookies()) != 1 || recorder.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatal("logout error response did not expire the cookie")
	}
}

func performHandlerRequest(method, path, body string, handler gin.HandlerFunc, user *service.AuthenticatedUser, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path, func(c *gin.Context) {
		if user != nil {
			SetAuthenticatedUser(c, *user)
		}
		handler(c)
	})
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		if cookie != nil {
			request.AddCookie(cookie)
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, wantStatus, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":{"code":"`+wantCode+`","message":"`) {
		t.Fatalf("body does not use common error format: %s", recorder.Body.String())
	}
}
