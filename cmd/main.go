package main

import (
	"backend/internal/app"
	"backend/internal/config"
	"backend/internal/utils"
	"log"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer logger.Sync()

	utils.LoadEnv(logger)

	cfg := config.LoadConfig()

	logger.Info("Config loaded",
		zap.String("server_port", cfg.ServerPort),
		zap.String("db_host", cfg.DBHost),
		zap.String("redis_url", cfg.RedisURL),
		zap.String("env", cfg.Env),
	)

	application, err := app.Bootstrap(&cfg, logger)
	if err != nil {
		logger.Fatal("Failed to bootstrap application", zap.Error(err))
	}

	logger.Info("Server started", zap.String("addr", "localhost:"+cfg.ServerPort))

	if err := application.Router.Serve(":" + cfg.ServerPort); err != nil {
		logger.Fatal("Server stopped with error", zap.Error(err))
	}
}
