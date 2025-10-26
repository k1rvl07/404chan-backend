package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/app"
	"backend/internal/config"
	"backend/internal/utils"

	"go.uber.org/zap"
)

func main() {
	logger, err := utils.NewLogger()
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

	addr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:    addr,
		Handler: application.Router.Engine,
	}

	go func() {
		logger.Info("Server started", zap.String("addr", "localhost"+addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server stopped with error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}
