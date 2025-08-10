package router

import (
	"backend/internal/app/health"
	"backend/internal/app/session"
	"backend/internal/app/user"
	"backend/internal/gateways/websocket"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
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

func (r *Router) RegisterHealthRoutes(ctrl *health.HealthController) {
	health.RegisterRoutes(r.Engine.Group("/api"), ctrl)
}

func (r *Router) RegisterWebSocketRoutes(hub *websocket.Hub) {
	websocket.RegisterRoutes(r.Engine, hub)
}

func (r *Router) RegisterSessionRoutes(service session.Service) {
	session.RegisterRoutes(r.Engine.Group("/api"), service)
}

func (r *Router) RegisterUserRoutes(handler user.Handler) {
	user.RegisterRoutes(r.Engine.Group("/api"), handler)
}

func (r *Router) Serve(addr string) error {
	return r.Engine.Run(addr)
}
