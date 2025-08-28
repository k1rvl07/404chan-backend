package thread

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	threads := rg.Group("/threads")
	{
		threads.POST("/:board_id", handler.CreateThread)
		threads.GET("/:board_id", handler.GetThreadsByBoardID)
		threads.GET("/cooldown", handler.GetThreadCooldown)
		threads.GET("/thread/:id", handler.GetThreadByID)
		threads.GET("/top", handler.GetTopThreads)
	}
}
