package app

import (
	"backend/internal/app/health"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/gateways/websocket"
	"backend/internal/providers/redis"
	"backend/internal/router"
	"backend/internal/utils"

	"go.uber.org/zap"
)

type Application struct {
	Router *router.Router
}

func Bootstrap(cfg *config.Config, logger *zap.Logger) (*Application, error) {

	db, err := db.Connect(cfg, logger)
	if err != nil {
		return nil, err
	}

	redisProvider := redis.NewRedisProvider(cfg.RedisURL, logger)

	hub := websocket.NewHub(logger)
	go hub.Run()

	healthChecker := utils.HealthChecker{
		DB:    db,
		Redis: redisProvider.Client,
	}
	healthController := health.NewHealthController(&healthChecker)

	r := router.NewRouter(logger)

	r.RegisterHealthRoutes(healthController)
	r.RegisterWebSocketRoutes(hub)

	return &Application{
		Router: r,
	}, nil
}
