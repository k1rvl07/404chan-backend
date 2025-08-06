package app

import (
    "backend/internal/app/health"
    "backend/internal/app/session"
    "backend/internal/config"
    "backend/internal/db"
    "backend/internal/gateways/websocket"
    "backend/internal/providers/redis"
    "backend/internal/router"
    "backend/internal/utils"

    "gorm.io/gorm"
    "go.uber.org/zap"
)

type Application struct {
    Router *router.Router
    DB     *gorm.DB
}

func Bootstrap(cfg *config.Config, logger *zap.Logger) (*Application, error) {
    dbConn, err := db.Connect(cfg, logger)
    if err != nil {
        return nil, err
    }

    redisProvider := redis.NewRedisProvider(cfg.RedisURL, logger)

    sessionRepo := session.NewRepository(dbConn)
    sessionService := session.NewService(sessionRepo)

    hub := websocket.NewHub(logger, sessionService)
    go hub.Run()

    healthChecker := utils.HealthChecker{
        DB:    dbConn,
        Redis: redisProvider.Client,
    }
    healthController := health.NewHealthController(&healthChecker)

    r := router.NewRouter(logger)

    r.RegisterHealthRoutes(healthController)
    r.RegisterWebSocketRoutes(hub)
    r.RegisterSessionRoutes(sessionService)

    return &Application{
        Router: r,
        DB:     dbConn,
    }, nil
}