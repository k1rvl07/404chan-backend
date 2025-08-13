package app

import (
	"backend/internal/app/board"
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

	redisProvider := redis.NewRedisProvider(cfg.RedisURL, logger, cfg.RedisTTL)
	eventBus := utils.NewEventBus()

	sessionRepo := session.NewRepository(dbConn)
	userRepo := user.NewRepository(dbConn)
	boardRepo := board.NewRepository(dbConn)

	sessionService := session.NewService(sessionRepo, redisProvider)
	userService := user.NewService(userRepo)
	boardService := board.NewService(boardRepo)

	hub := websocket.NewHub(logger, sessionService, eventBus, userRepo, redisProvider)
	go hub.Run()

	healthHandler := health.NewHandler(&utils.HealthChecker{
		DB:    dbConn,
		Redis: redisProvider.Client,
	})
	sessionHandler := session.NewHandler(sessionService)
	userHandler := user.NewHandler(userService, sessionService, eventBus, logger, redisProvider)
	boardHandler := board.NewHandler(boardService)

	r := router.NewRouter(logger)

	r.RegisterHealthRoutes(healthHandler)
	r.RegisterWebSocketRoutes(hub)
	r.RegisterSessionRoutes(sessionHandler)
	r.RegisterUserRoutes(userHandler)
	r.RegisterBoardRoutes(boardHandler)

	return &Application{
		Router: r,
		DB:     dbConn,
	}, nil
}
