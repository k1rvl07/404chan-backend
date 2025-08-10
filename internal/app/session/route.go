package session

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg gin.IRoutes, service Service) {
	handler := NewHandler(service)
	rg.POST("/session", handler.CreateSession)
}
