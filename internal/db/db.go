package db

import (
	"backend/internal/app/attachment"
	"backend/internal/app/board"
	"backend/internal/app/message"
	"backend/internal/app/session"
	"backend/internal/app/thread"
	"backend/internal/app/user"
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

func Migrate(db *gorm.DB, logger *zap.Logger) error {
	logger.Info("Running database migrations...")

	err := db.AutoMigrate(
		&user.User{},
		&user.UserActivity{},
		&session.Session{},
		&board.Board{},
		&thread.Thread{},
		&thread.ThreadActivity{},
		&message.Message{},
		&attachment.Attachment{},
	)
	if err != nil {
		logger.Error("Migrations failed", zap.Error(err))
		return err
	}

	logger.Info("Database migrations completed successfully")
	return nil
}
