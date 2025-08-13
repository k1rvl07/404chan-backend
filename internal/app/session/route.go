package session

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg gin.IRoutes, handler Handler) {
	rg.POST("/session", handler.CreateSession)
}
