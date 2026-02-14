package attachment

import "github.com/gin-gonic/gin"

// RegisterRoutes registers attachment routes
// @Summary Attachment routes
// @Description Routes for attachment management
// @Tags Attachment
func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	rg.GET("/attachments", handler.GetAttachments)
	rg.DELETE("/attachments", handler.DeleteTemporary)
}
