package router

import (
	"backend/internal/app/health"
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
	health.RegisterRoutes(r.Engine, ctrl)
}

func (r *Router) RegisterWebSocketRoutes(hub *websocket.Hub) {
	websocket.RegisterRoutes(r.Engine, hub)
}

func (r *Router) Serve(addr string) error {
	return r.Engine.Run(addr)
}
