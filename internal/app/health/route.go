package health

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg gin.IRoutes, handler Handler) {
	rg.GET("/health", handler.Check)
}
