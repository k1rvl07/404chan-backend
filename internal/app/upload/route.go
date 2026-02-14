package upload

import "github.com/gin-gonic/gin"

// RegisterRoutes registers upload routes
// @Summary Upload routes
// @Description Routes for file uploads
// @Tags Upload
func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	rg.POST("/upload", handler.Upload)
	rg.POST("/upload/confirm", handler.ConfirmFiles)
}
