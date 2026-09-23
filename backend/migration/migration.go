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

		if err := seedMasterData(tx); err != nil {
			return err
		}

		return nil
	})
}
