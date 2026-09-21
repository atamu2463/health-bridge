package config

import (
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectDB(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// 接続文字列や認証情報がエラー本文へ含まれる可能性があるため、詳細は返さない。
		return nil, errors.New("PostgreSQLへの接続を確認できません")
	}

	return db, nil
}
