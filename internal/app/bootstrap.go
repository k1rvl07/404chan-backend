package app

import (
	"backend/internal/app/attachment"
	"backend/internal/app/board"
	"backend/internal/app/health"
	"backend/internal/app/message"
	"backend/internal/app/session"
	"backend/internal/app/thread"
	"backend/internal/app/upload"
	"backend/internal/app/user"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/db/seeder"
	"backend/internal/gateways/websocket"
	"backend/internal/providers/minio"
	"backend/internal/providers/redis"
	"backend/internal/router"
	"backend/internal/utils"
	"time"

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

	if err := db.Migrate(dbConn, logger); err != nil {
		return nil, err
	}

	seed := seeder.NewSeeder(dbConn, logger)
	if err := seed.Seed(); err != nil {
		logger.Warn("Failed to run seeders", zap.Error(err))
	}

	redisProvider := redis.NewRedisProvider(cfg.RedisURL, logger, cfg.RedisTTL)
	minioProvider, err := minio.NewMinioProvider(cfg, logger)
	if err != nil {
		logger.Warn("Failed to initialize MinIO provider", zap.Error(err))
		minioProvider = nil
	}
	eventBus := utils.NewEventBus()

	sessionRepo := session.NewRepository(dbConn)
	userRepo := user.NewRepository(dbConn)
	boardRepo := board.NewRepository(dbConn)
	threadRepo := thread.NewRepository(dbConn)
	messageRepo := message.NewRepository(dbConn)
	attachmentRepo := attachment.NewRepository(dbConn)

	attachmentService := attachment.NewService(attachmentRepo, dbConn, minioProvider, logger)
	uploadHandler := upload.NewHandler(minioProvider, attachmentService, logger)

	sessionService := session.NewService(sessionRepo, redisProvider)
	userService := user.NewService(userRepo, sessionService, redisProvider, logger)
	boardService := board.NewService(boardRepo)
	threadService := thread.NewService(threadRepo, sessionService, userService, dbConn, redisProvider, eventBus, logger, minioProvider, attachmentService)
	messageService := message.NewService(messageRepo, sessionService, threadService, dbConn, redisProvider, eventBus, logger, minioProvider, attachmentService)

	hub := websocket.NewHub(logger, sessionService, eventBus, userRepo, redisProvider)
	go hub.Run()

	if minioProvider != nil {
		go func() {
			ticker := time.NewTicker(15 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				if err := minioProvider.DeleteTmpFilesOlderThan(1 * time.Hour); err != nil {
					logger.Warn("Failed to cleanup old tmp files", zap.Error(err))
				}
			}
		}()
	}

	healthHandler := health.NewHandler(&utils.HealthChecker{
		DB:    dbConn,
		Redis: redisProvider.Client,
	})
	sessionHandler := session.NewHandler(sessionService)
	userHandler := user.NewHandler(userService, sessionService, eventBus, logger, redisProvider)
	boardHandler := board.NewHandler(boardService)
	threadHandler := thread.NewHandler(threadService, sessionService, userService)
	messageHandler := message.NewHandler(messageService, sessionService)
	attachmentHandler := attachment.NewHandler(attachmentService)

	r := router.NewRouter(logger)

	r.RegisterHealthRoutes(healthHandler)
	r.RegisterWebSocketRoutes(hub)
	r.RegisterSessionRoutes(sessionHandler)
	r.RegisterUserRoutes(userHandler)
	r.RegisterBoardRoutes(boardHandler)
	r.RegisterThreadRoutes(threadHandler)
	r.RegisterMessageRoutes(messageHandler)
	r.RegisterAttachmentRoutes(attachmentHandler)
	r.RegisterUploadRoutes(uploadHandler)
	r.RegisterSwaggerRoutes()

	return &Application{
		Router: r,
		DB:     dbConn,
	}, nil
}
