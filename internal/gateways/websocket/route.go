package websocket

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg gin.IRoutes, hub *Hub) {
	rg.GET("/ws", hub.ServeWS)
}
