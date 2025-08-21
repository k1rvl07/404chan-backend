package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	users := rg.Group("/user")
	{
		users.GET("", handler.GetUser)
		users.PATCH("/nickname", handler.UpdateNickname)
		users.GET("/cooldown", handler.GetCooldown)
	}
}
