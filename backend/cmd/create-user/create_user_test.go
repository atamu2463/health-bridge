package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"backend/config"
	"backend/migration"
	"backend/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestValidateCreateUserInput(t *testing.T) {
	dynamicPassword := fmt.Sprintf("Validation-%d!", time.Now().UnixNano())
	tests := []struct {
		name  string
		input createUserInput
	}{
		{name: "名前が空", input: createUserInput{Email: "user@example.invalid", Role: model.RoleNameManager, Password: dynamicPassword}},
		{name: "メールが空", input: createUserInput{Name: "利用者", Role: model.RoleNameManager, Password: dynamicPassword}},
		{name: "メール形式が不正", input: createUserInput{Name: "利用者", Email: "invalid", Role: model.RoleNameManager, Password: dynamicPassword}},
		{name: "roleが不正", input: createUserInput{Name: "利用者", Email: "user@example.invalid", Role: "admin", Password: dynamicPassword}},
		{name: "パスワードが空", input: createUserInput{Name: "利用者", Email: "user@example.invalid", Role: model.RoleNameManager}},
		{name: "employeeのmanagerが空", input: createUserInput{Name: "利用者", Email: "user@example.invalid", Role: model.RoleNameEmployee, Password: dynamicPassword}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := validateCreateUserInput(tt.input); err == nil {
				t.Fatalf("validateCreateUserInput() error = nil")
			}
		})
	}
}

func TestReadAndConfirmPasswordDoesNotEchoInput(t *testing.T) {
	password := fmt.Sprintf("Hidden-%d!", time.Now().UnixNano())
	input, err := os.CreateTemp(t.TempDir(), "password-input")
	if err != nil {
		t.Fatalf("一時入力ファイルを作成できませんでした: %v", err)
	}
	if _, err := input.WriteString(password + "\n" + password + "\n"); err != nil {
		t.Fatalf("一時入力を書き込めませんでした: %v", err)
	}
	if _, err := input.Seek(0, 0); err != nil {
		t.Fatalf("一時入力を先頭へ戻せませんでした: %v", err)
	}

	var output bytes.Buffer
	got, err := readAndConfirmPassword(input, &output)
	if err != nil {
		t.Fatalf("readAndConfirmPassword() error = %v", err)
	}
	if got != password {
		t.Fatal("readAndConfirmPassword()が入力したパスワードを返しませんでした")
	}
	if strings.Contains(output.String(), password) {
		t.Fatal("パスワードが出力へ含まれました")
	}
}

func TestCreateUserOnPostgreSQL(t *testing.T) {
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

	schemaName := fmt.Sprintf("issue139_user_test_%d", time.Now().UnixNano())
	rollbackError := errors.New("テスト用トランザクションをロールバックします")
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

		suffix := fmt.Sprintf("%d", time.Now().UnixNano())
		managerEmail := "manager-" + suffix + "@example.invalid"
		employeeEmail := "employee-" + suffix + "@example.invalid"
		managerPassword := "Manager-" + suffix + "!"
		employeePassword := "Employee-" + suffix + "!"

		manager, err := createUser(tx, createUserInput{
			Name:     "テスト管理者",
			Email:    managerEmail,
			Role:     model.RoleNameManager,
			Password: managerPassword,
		}, bcrypt.MinCost)
		if err != nil {
			return fmt.Errorf("managerを作成できませんでした: %w", err)
		}
		if manager.PasswordHash == managerPassword {
			return errors.New("managerのパスワードが平文で保存されました")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(manager.PasswordHash), []byte(managerPassword)); err != nil {
			return errors.New("managerのbcryptパスワードを照合できませんでした")
		}
		var storedManager model.User
		if err := tx.Preload("Role").First(&storedManager, manager.ID).Error; err != nil {
			return err
		}
		if storedManager.Role.Name != model.RoleNameManager {
			return errors.New("managerへmanager roleが保存されませんでした")
		}

		employee, err := createUser(tx, createUserInput{
			Name:         "テスト従業員",
			Email:        employeeEmail,
			Role:         model.RoleNameEmployee,
			ManagerEmail: managerEmail,
			Password:     employeePassword,
		}, bcrypt.MinCost)
		if err != nil {
			return fmt.Errorf("employeeを作成できませんでした: %w", err)
		}
		if employee.ManagerID == nil || *employee.ManagerID != manager.ID {
			return errors.New("employeeへmanagerが設定されませんでした")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(employeePassword)); err != nil {
			return errors.New("employeeのbcryptパスワードを照合できませんでした")
		}
		var storedEmployee model.User
		if err := tx.Preload("Role").First(&storedEmployee, employee.ID).Error; err != nil {
			return err
		}
		if storedEmployee.Role.Name != model.RoleNameEmployee {
			return errors.New("employeeへemployee roleが保存されませんでした")
		}

		_, duplicateErr := createUser(tx, createUserInput{
			Name:     "重複確認",
			Email:    managerEmail,
			Role:     model.RoleNameManager,
			Password: employeePassword,
		}, bcrypt.MinCost)
		if !errors.Is(duplicateErr, errEmailAlreadyExists) {
			return fmt.Errorf("重複メールエラー = %v", duplicateErr)
		}

		for _, secret := range []string{
			managerPassword,
			employeePassword,
			manager.PasswordHash,
			databaseURL,
		} {
			if secret != "" && strings.Contains(duplicateErr.Error(), secret) {
				return errors.New("エラーへ秘密情報が含まれています")
			}
		}

		return rollbackError
	})
	if !errors.Is(err, rollbackError) {
		t.Fatalf("PostgreSQL統合テストに失敗しました: %v", err)
	}
}
