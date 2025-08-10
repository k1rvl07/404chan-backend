package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg gin.IRoutes, handler Handler) {
	rg.GET("/user", handler.GetUser)
	rg.PATCH("/user/nickname", handler.UpdateNickname)
}
