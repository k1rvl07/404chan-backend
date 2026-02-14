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

// @Summary Get all boards
// @Description Get a list of all available boards
// @Tags Board
// @Accept json
// @Produce json
// @Success 200 {object} BoardListResponse
// @Router /api/boards [get]
func (h *handler) GetAllBoards(c *gin.Context) {
	boards, err := h.service.GetAllBoards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch boards"})
		return
	}
	c.JSON(http.StatusOK, BoardListResponse{Boards: boards})
}

// @Summary Get board by slug
// @Description Get a specific board by its slug identifier
// @Tags Board
// @Accept json
// @Produce json
// @Param slug path string true "Board slug"
// @Success 200 {object} Board
// @Failure 404 {object} ErrorResponse
// @Router /api/boards/{slug} [get]
func (h *handler) GetBoardBySlug(c *gin.Context) {
	slug := c.Param("slug")
	board, err := h.service.GetBoardBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "board not found"})
		return
	}
	c.JSON(http.StatusOK, board)
}
