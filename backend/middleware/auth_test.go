package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/api"
	"backend/handler"
	"backend/service"

	"github.com/gin-gonic/gin"
)

type fakeAuthenticator struct {
	user  service.AuthenticatedUser
	err   error
	token string
}

func (a *fakeAuthenticator) Authenticate(_ context.Context, token string) (service.AuthenticatedUser, error) {
	a.token = token
	return a.user, a.err
}

func TestRequireAuthentication(t *testing.T) {
	tests := []struct {
		name       string
		cookie     *http.Cookie
		err        error
		wantStatus int
		wantCode   string
		wantNext   bool
	}{
		{name: "success", cookie: &http.Cookie{Name: handler.SessionCookieName, Value: "token"}, wantStatus: http.StatusOK, wantNext: true},
		{name: "missing cookie", wantStatus: http.StatusUnauthorized, wantCode: api.ErrorCodeUnauthenticated},
		{name: "invalid session", cookie: &http.Cookie{Name: handler.SessionCookieName, Value: "invalid"}, err: service.ErrUnauthenticated, wantStatus: http.StatusUnauthorized, wantCode: api.ErrorCodeUnauthenticated},
		{name: "internal error", cookie: &http.Cookie{Name: handler.SessionCookieName, Value: "token"}, err: service.ErrInternal, wantStatus: http.StatusInternalServerError, wantCode: api.ErrorCodeInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authenticator := &fakeAuthenticator{user: service.AuthenticatedUser{ID: 1, Role: "manager"}, err: tt.err}
			nextCalled := false
			recorder := performMiddlewareRequest([]gin.HandlerFunc{RequireAuthentication(authenticator)}, func(c *gin.Context) {
				nextCalled = true
				if _, ok := handler.AuthenticatedUser(c); !ok {
					t.Fatal("authenticated user is missing from context")
				}
				c.Status(http.StatusOK)
			}, tt.cookie)
			if recorder.Code != tt.wantStatus || nextCalled != tt.wantNext {
				t.Fatalf("status=%d next=%v, want status=%d next=%v", recorder.Code, nextCalled, tt.wantStatus, tt.wantNext)
			}
			if tt.wantCode != "" && !strings.Contains(recorder.Body.String(), `"code":"`+tt.wantCode+`"`) {
				t.Fatalf("unexpected error body: %s", recorder.Body.String())
			}
		})
	}
}

func TestRequireRoleDistinguishesUnauthenticatedAndForbidden(t *testing.T) {
	tests := []struct {
		name       string
		user       *service.AuthenticatedUser
		wantStatus int
		wantCode   string
	}{
		{name: "allowed", user: &service.AuthenticatedUser{Role: "manager"}, wantStatus: http.StatusOK},
		{name: "role missing", user: &service.AuthenticatedUser{Role: "employee"}, wantStatus: http.StatusForbidden, wantCode: api.ErrorCodeForbidden},
		{name: "not authenticated", wantStatus: http.StatusUnauthorized, wantCode: api.ErrorCodeUnauthenticated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setUser := func(c *gin.Context) {
				if tt.user != nil {
					handler.SetAuthenticatedUser(c, *tt.user)
				}
				c.Next()
			}
			recorder := performMiddlewareRequest([]gin.HandlerFunc{setUser, RequireRole("manager")}, func(c *gin.Context) { c.Status(http.StatusOK) }, nil)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status=%d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantCode != "" && !strings.Contains(recorder.Body.String(), `"code":"`+tt.wantCode+`"`) {
				t.Fatalf("unexpected error body: %s", recorder.Body.String())
			}
		})
	}
}

func performMiddlewareRequest(middlewares []gin.HandlerFunc, endpoint gin.HandlerFunc, cookie *http.Cookie) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middlewares...)
	router.GET("/test", endpoint)
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
