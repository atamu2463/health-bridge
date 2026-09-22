package main

import (
	"fmt"
	"log"
	"os"

	"backend/config"
	"backend/migration"
)

func main() {
	if err := runMigration(); err != nil {
		log.Printf("マイグレーションに失敗しました: %v", err)
		os.Exit(1)
	}

	log.Print("マイグレーションとマスターデータ投入が完了しました")
}

func runMigration() (runErr error) {
	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		return err
	}

	db, err := config.ConnectDB(databaseURL)
	if err != nil {
		return fmt.Errorf("DB接続に失敗しました: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("DBインスタンスの取得に失敗しました: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil && runErr == nil {
			runErr = fmt.Errorf("DB接続の終了に失敗しました: %w", err)
		}
	}()

	return migration.Run(db)
}
