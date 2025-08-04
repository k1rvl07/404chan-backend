package utils

import (
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func LoadEnv(logger *zap.Logger) {
	if err := godotenv.Load(); err != nil {
		logger.Warn("ENV file not found or failed to load, using defaults")
	} else {
		logger.Info("ENV file loaded successfully")
	}
}
