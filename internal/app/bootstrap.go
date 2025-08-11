package app

import (
	"backend/internal/app/health"
	"backend/internal/app/session"
	"backend/internal/app/user"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/gateways/websocket"
	"backend/internal/providers/redis"
	"backend/internal/router"
	"backend/internal/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
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
	eventBus := utils.NewEventBus()

	sessionRepo := session.NewRepository(dbConn)
	sessionService := session.NewService(sessionRepo)

	userRepo := user.NewRepository(dbConn)
	userService := user.NewService(userRepo)

	hub := websocket.NewHub(logger, sessionService, eventBus, userRepo)
	go hub.Run()

	userHandler := user.NewHandler(userService, sessionService, eventBus, logger)

	healthChecker := utils.HealthChecker{
		DB:    dbConn,
		Redis: redisProvider.Client,
	}
	healthController := health.NewHealthController(&healthChecker)

	r := router.NewRouter(logger)

	r.RegisterHealthRoutes(healthController)
	r.RegisterWebSocketRoutes(hub)
	r.RegisterSessionRoutes(sessionService)
	r.RegisterUserRoutes(userHandler)

	return &Application{
		Router: r,
		DB:     dbConn,
	}, nil
}
