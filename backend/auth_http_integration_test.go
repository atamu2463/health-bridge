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
	"backend/migration"
	"backend/model"
	"backend/repository"
	"backend/service"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestAuthenticationHTTPFlowOnPostgreSQL(t *testing.T) {
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
	schemaName := fmt.Sprintf("issue140_auth_http_%d", time.Now().UnixNano())
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
		if err := tx.Where("name = ?", model.RoleNameEmployee).First(&role).Error; err != nil {
			return err
		}
		password := "integration-password"
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			return err
		}
		user := model.User{
			Name:         "HTTP統合テスト従業員",
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
			`{"email":"http-integration@example.invalid","password":"integration-password"}`,
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
		if meResponse.Code != http.StatusOK || !strings.Contains(meResponse.Body.String(), `"role":"employee"`) {
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

		return rollbackError
	})
	if !errors.Is(err, rollbackError) {
		t.Fatalf("PostgreSQL HTTP統合テストに失敗しました: %v", err)
	}
}
