package board

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	GetAllBoards(c *gin.Context)
	GetBoardBySlug(c *gin.Context)
}

type handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return &handler{service: service}
}

func (h *handler) GetAllBoards(c *gin.Context) {
	boards, err := h.service.GetAllBoards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch boards"})
		return
	}
	c.JSON(http.StatusOK, boards)
}

func (h *handler) GetBoardBySlug(c *gin.Context) {
	slug := c.Param("slug")
	board, err := h.service.GetBoardBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
		return
	}
	c.JSON(http.StatusOK, board)
}
