package migration

import (
	"fmt"

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
	var hasCollision bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM users
			GROUP BY LOWER(BTRIM(email))
			HAVING COUNT(*) > 1
		)
	`).Scan(&hasCollision).Error; err != nil {
		return fmt.Errorf("メールアドレス正規化前の衝突検査に失敗しました: %w", err)
	}
	if hasCollision {
		return fmt.Errorf("メールアドレスの正規化結果が重複するユーザーが存在します")
	}

	if err := db.Exec(`
		UPDATE users
		SET email = LOWER(BTRIM(email))
		WHERE email IS DISTINCT FROM LOWER(BTRIM(email))
	`).Error; err != nil {
		return fmt.Errorf("メールアドレスの正規化に失敗しました: %w", err)
	}

	return nil
}
