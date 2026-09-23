package migration

import (
	"fmt"
	"strings"

	"backend/model"

	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(
			&model.Role{},
			&model.User{},
			&model.Session{},
			&model.Condition{},
			&model.HealthRecord{},
		); err != nil {
			return fmt.Errorf("テーブルのマイグレーションに失敗しました: %w", err)
		}

		if err := normalizeUserEmails(tx); err != nil {
			return err
		}

		if err := seedMasterData(tx); err != nil {
			return err
		}

		return nil
	})
}

func normalizeUserEmails(db *gorm.DB) error {
	if err := db.Exec("LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
		return fmt.Errorf("メールアドレス正規化前のusersテーブルロックに失敗しました: %w", err)
	}

	type userEmail struct {
		ID    uint
		Email string
	}
	var userEmails []userEmail
	if err := db.Model(&model.User{}).
		Select("id", "email").
		Order("id").
		Find(&userEmails).Error; err != nil {
		return fmt.Errorf("メールアドレス正規化対象の取得に失敗しました: %w", err)
	}

	type normalizedUserEmail struct {
		ID       uint
		Original string
		Value    string
	}
	normalizedEmails := make([]normalizedUserEmail, len(userEmails))
	for index, user := range userEmails {
		normalizedEmails[index] = normalizedUserEmail{
			ID:       user.ID,
			Original: user.Email,
			Value:    strings.ToLower(strings.TrimSpace(user.Email)),
		}
	}

	seen := make(map[string]struct{}, len(normalizedEmails))
	for _, email := range normalizedEmails {
		if _, exists := seen[email.Value]; exists {
			return fmt.Errorf("メールアドレスの正規化結果が重複するユーザーが存在します")
		}
		seen[email.Value] = struct{}{}
	}

	for _, email := range normalizedEmails {
		if email.Original == email.Value {
			continue
		}
		result := db.Model(&model.User{}).
			Where("id = ?", email.ID).
			UpdateColumn("email", email.Value)
		if result.Error != nil {
			return fmt.Errorf("メールアドレスの正規化に失敗しました: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("メールアドレスの正規化対象を更新できませんでした")
		}
	}

	return nil
}
