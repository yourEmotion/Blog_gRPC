package config

import (
	"fmt"
	"os"

	"github.com/yourEmotion/Blog_gRPC/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitPostgres() (*gorm.DB, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "host=localhost user=ermachine dbname=blog port=5432 sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Авто-миграция модели Post
	if err := db.AutoMigrate(&models.Post{}); err != nil {
		return nil, fmt.Errorf("failed migrate: %w", err)
	}

	return db, nil
}
