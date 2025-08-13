package board

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg gin.IRoutes, handler Handler) {
	rg.GET("/boards", handler.GetAllBoards)
	rg.GET("/boards/:slug", handler.GetBoardBySlug)
}
