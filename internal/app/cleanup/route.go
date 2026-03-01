package cleanup

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	cleanup := rg.Group("/cleanup")
	{
		cleanup.POST("", handler.Cleanup)
	}
}
