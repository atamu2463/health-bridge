package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"backend/config"
	"backend/handler"
	"backend/middleware"
	"backend/migration"
	"backend/model"
	"backend/repository"
	"backend/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestAuthenticationHTTPFlowOnPostgreSQL(t *testing.T) {
	for _, role := range []string{model.RoleNameEmployee, model.RoleNameManager} {
		t.Run(role, func(t *testing.T) { testAuthenticationHTTPFlow(t, role) })
	}
}

func testAuthenticationHTTPFlow(t *testing.T, roleName string) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URLが未設定のためPostgreSQL統合テストをスキップします")
	}

	db, err := config.ConnectDB(databaseURL)
	if err != nil {
		t.Fatalf("テストDBへの接続に失敗しました: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("テストDBインスタンスの取得に失敗しました: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("テストDB接続の終了に失敗しました: %v", err)
		}
	})

	rollbackError := errors.New("テスト用トランザクションをロールバックします")
	schemaName := fmt.Sprintf("issue149_auth_http_%d", time.Now().UnixNano())
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE SCHEMA "` + schemaName + `"`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`SET LOCAL search_path TO "` + schemaName + `"`).Error; err != nil {
			return err
		}
		if err := migration.Run(tx); err != nil {
			return err
		}

		var role model.Role
		if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
			return err
		}
		password := "integration-password"
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			return err
		}
		user := model.User{
			Name:         "HTTP統合テストユーザー",
			Email:        "http-integration@example.invalid",
			PasswordHash: string(passwordHash),
			RoleID:       role.ID,
			IsActive:     true,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		authRepository := repository.NewAuthRepository(tx)
		authService := service.NewAuthService(authRepository)
		router := newRouter([]string{testAllowedOrigin}, authService, handler.NewAuthHandler(authService, false), false)

		loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(
			fmt.Sprintf(`{"email":"http-integration@example.invalid","password":"integration-password","expectedRole":%q}`, roleName),
		))
		loginRequest.Header.Set("Content-Type", "application/json")
		loginRequest.Header.Set("Origin", testAllowedOrigin)
		loginResponse := httptest.NewRecorder()
		router.ServeHTTP(loginResponse, loginRequest)
		if loginResponse.Code != http.StatusOK {
			return fmt.Errorf("login status = %d: %s", loginResponse.Code, loginResponse.Body.String())
		}
		cookies := loginResponse.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != handler.SessionCookieName {
			return errors.New("login responseにsession Cookieがありません")
		}
		sessionCookie := cookies[0]

		meRequest := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		meRequest.AddCookie(sessionCookie)
		meResponse := httptest.NewRecorder()
		router.ServeHTTP(meResponse, meRequest)
		if meResponse.Code != http.StatusOK || !strings.Contains(meResponse.Body.String(), `"role":"`+roleName+`"`) {
			return fmt.Errorf("me response = %d: %s", meResponse.Code, meResponse.Body.String())
		}

		logoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		logoutRequest.Header.Set("Origin", testAllowedOrigin)
		logoutRequest.AddCookie(sessionCookie)
		logoutResponse := httptest.NewRecorder()
		router.ServeHTTP(logoutResponse, logoutRequest)
		if logoutResponse.Code != http.StatusNoContent {
			return fmt.Errorf("logout status = %d: %s", logoutResponse.Code, logoutResponse.Body.String())
		}

		revokedRequest := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		revokedRequest.AddCookie(sessionCookie)
		revokedResponse := httptest.NewRecorder()
		router.ServeHTTP(revokedResponse, revokedRequest)
		if revokedResponse.Code != http.StatusUnauthorized {
			return fmt.Errorf("revoked me status = %d: %s", revokedResponse.Code, revokedResponse.Body.String())
		}

		secondLogoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		secondLogoutRequest.Header.Set("Origin", testAllowedOrigin)
		secondLogoutRequest.AddCookie(sessionCookie)
		secondLogoutResponse := httptest.NewRecorder()
		router.ServeHTTP(secondLogoutResponse, secondLogoutRequest)
		if secondLogoutResponse.Code != http.StatusNoContent {
			return fmt.Errorf("second logout status = %d", secondLogoutResponse.Code)
		}

		oppositeRole := model.RoleNameManager
		if roleName == oppositeRole {
			oppositeRole = model.RoleNameEmployee
		}
		protected := router.Group("/api/test", middleware.RequireAuthentication(authService))
		protected.GET("/same", middleware.RequireRole(roleName), func(c *gin.Context) { c.Status(http.StatusNoContent) })
		protected.GET("/other", middleware.RequireRole(oppositeRole), func(c *gin.Context) { c.Status(http.StatusNoContent) })
		for _, wrongPassword := range []bool{false, true} {
			for _, withCookie := range []bool{false, true} {
				// 独立した試行としてレート制限のカウンタを分離する。
				attemptRouter := newRouter([]string{testAllowedOrigin}, authService, handler.NewAuthHandler(authService, false), false)
				fresh, err := authService.Login(t.Context(), user.Email, password, roleName)
				if err != nil {
					return err
				}
				oldCookie := &http.Cookie{Name: handler.SessionCookieName, Value: fresh.Token}
				for path, status := range map[string]int{"/api/test/same": http.StatusNoContent, "/api/test/other": http.StatusForbidden} {
					request := httptest.NewRequest(http.MethodGet, path, nil)
					request.AddCookie(oldCookie)
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					if response.Code != status {
						return fmt.Errorf("protected route status = %d, want %d", response.Code, status)
					}
				}
				if !withCookie {
					if err := authService.Logout(t.Context(), fresh.Token); err != nil {
						return err
					}
				}
				attemptPassword, expectedRole := password, oppositeRole
				if wrongPassword {
					attemptPassword, expectedRole = "wrong-password", roleName
				}
				body := fmt.Sprintf(`{"email":%q,"password":%q,"expectedRole":%q}`, user.Email, attemptPassword, expectedRole)
				request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Origin", testAllowedOrigin)
				if withCookie {
					request.AddCookie(oldCookie)
				}
				response := httptest.NewRecorder()
				attemptRouter.ServeHTTP(response, request)
				if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"code":"invalid_credentials"`) {
					return fmt.Errorf("role mismatch status = %d: %s", response.Code, response.Body.String())
				}
				for _, secret := range []string{user.Email, password, fresh.Token, roleName, oppositeRole} {
					if strings.Contains(response.Body.String(), secret) {
						return errors.New("role mismatch response contains sensitive information")
					}
				}
				cookies := response.Result().Cookies()
				if withCookie && (len(cookies) != 1 || cookies[0].MaxAge >= 0 || cookies[0].Value != "") {
					return errors.New("role mismatch did not expire existing cookie")
				}
				if !withCookie && len(cookies) != 0 {
					return errors.New("role mismatch issued a cookie")
				}
				var count int64
				if err := tx.Model(&model.Session{}).Count(&count).Error; err != nil {
					return err
				}
				if count != 0 {
					return fmt.Errorf("role mismatch left %d sessions", count)
				}
				replay := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
				replay.AddCookie(oldCookie)
				replayResponse := httptest.NewRecorder()
				attemptRouter.ServeHTTP(replayResponse, replay)
				if replayResponse.Code != http.StatusUnauthorized {
					return errors.New("old cookie still authenticates after mismatch")
				}
			}

		}
		return rollbackError
	})
	if !errors.Is(err, rollbackError) {
		t.Fatalf("PostgreSQL HTTP統合テストに失敗しました: %v", err)
	}
}
