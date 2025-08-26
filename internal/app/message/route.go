package message

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	messages := rg.Group("/messages")
	{
		messages.POST("/:thread_id", handler.CreateMessage)
		messages.GET("/:thread_id", handler.GetMessagesByThreadID)
		messages.GET("/cooldown", handler.GetMessageCooldown)
		messages.GET("/message/:id", handler.GetMessageByID)
	}
}
