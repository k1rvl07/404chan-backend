package router

import (
	"backend/internal/app/attachment"
	"backend/internal/app/board"
	"backend/internal/app/health"
	"backend/internal/app/message"
	"backend/internal/app/session"
	"backend/internal/app/thread"
	"backend/internal/app/upload"
	"backend/internal/app/user"
	"backend/internal/gateways/websocket"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(logger *zap.Logger) *Router {
	engine := gin.New()
	engine.Use(middleware.CORSMiddleware())
	engine.Use(middleware.LoggerMiddleware(logger))
	engine.Use(gin.Recovery())
	return &Router{Engine: engine}
}

func (r *Router) RegisterHealthRoutes(handler health.Handler) {
	health.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) RegisterWebSocketRoutes(hub *websocket.Hub) {
	websocket.RegisterRoutes(r.Engine, hub)
}

func (r *Router) RegisterSessionRoutes(handler session.Handler) {
	session.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) RegisterUserRoutes(handler user.Handler) {
	user.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) RegisterBoardRoutes(handler board.Handler) {
	board.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) RegisterThreadRoutes(handler thread.Handler) {
	thread.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) RegisterMessageRoutes(handler message.Handler) {
	message.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) RegisterAttachmentRoutes(handler attachment.Handler) {
	attachment.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) RegisterUploadRoutes(handler *upload.Handler) {
	upload.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) RegisterSwaggerRoutes() {
	r.Engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (r *Router) Serve(addr string) error {
	return r.Engine.Run(addr)
}
