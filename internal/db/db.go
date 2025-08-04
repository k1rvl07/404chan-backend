package db

import (
	"backend/internal/config"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	dsn := cfg.PostgresDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	logger.Info("Connected to PostgreSQL",
		zap.String("host", cfg.DBHost),
		zap.String("database", cfg.DBName),
	)

	return db, nil
}
