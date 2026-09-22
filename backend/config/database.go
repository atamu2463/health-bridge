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
		return nil, errors.New("PostgreSQLへの接続を確認できません")
	}

	return db, nil
}
