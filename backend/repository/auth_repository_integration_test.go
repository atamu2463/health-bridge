package repository_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"backend/config"
	"backend/migration"
	"backend/model"
	"backend/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestAuthRepositoryOnPostgreSQL(t *testing.T) {
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
	schemaName := fmt.Sprintf("issue140_auth_repository_%d", time.Now().UnixNano())
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
		if err := tx.Where("name = ?", model.RoleNameManager).First(&role).Error; err != nil {
			return err
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte("repository-test-password"), bcrypt.MinCost)
		if err != nil {
			return err
		}
		user := model.User{
			Name:         "Repositoryテスト管理者",
			Email:        "repository@example.invalid",
			PasswordHash: string(passwordHash),
			RoleID:       role.ID,
			IsActive:     true,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		authRepository := repository.NewAuthRepository(tx)
		foundUser, err := authRepository.FindUserByEmail(context.Background(), user.Email)
		if err != nil {
			return err
		}
		if foundUser.ID != user.ID || foundUser.Role.Name != model.RoleNameManager {
			return errors.New("取得したuserまたはroleが不正です")
		}

		digest := sha256.Sum256([]byte("repository-session-token"))
		session := model.Session{
			UserID:      user.ID,
			TokenDigest: digest[:],
			ExpiresAt:   time.Now().Add(time.Hour),
		}
		if err := authRepository.CreateSession(context.Background(), session); err != nil {
			return err
		}

		foundSession, err := authRepository.FindSessionByTokenDigest(context.Background(), digest[:])
		if err != nil {
			return err
		}
		if foundSession.User.ID != user.ID || foundSession.User.Role.Name != model.RoleNameManager {
			return errors.New("取得したsessionのuserまたはroleが不正です")
		}

		if err := authRepository.DeleteSessionByTokenDigest(context.Background(), digest[:]); err != nil {
			return err
		}
		if err := authRepository.DeleteSessionByTokenDigest(context.Background(), digest[:]); err != nil {
			return fmt.Errorf("削除済みsessionの再削除に失敗しました: %w", err)
		}
		if _, err := authRepository.FindSessionByTokenDigest(context.Background(), digest[:]); !errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("削除後の検索エラー = %v, want ErrNotFound", err)
		}

		return rollbackError
	})
	if !errors.Is(err, rollbackError) {
		t.Fatalf("PostgreSQL統合テストに失敗しました: %v", err)
	}
}
