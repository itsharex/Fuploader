package database

import (
	"Fuploader/internal/config"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() error {
	dbPath := config.GetDbPath()
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("create db dir failed: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("open db failed: %w", err)
	}

	DB = db
	return migrate()
}

func migrate() error {
	return DB.AutoMigrate(
		&Account{},
		&Video{},
		&UploadTask{},
		&ScheduleConfig{},
		&ScheduledTask{},
		&UploadLog{},
	)
}

func GetDB() *gorm.DB {
	return DB
}

func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
