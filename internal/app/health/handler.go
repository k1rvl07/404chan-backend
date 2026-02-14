package health

import (
	"net/http"

	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	Check(c *gin.Context)
}

type handler struct {
	checker *utils.HealthChecker
}

func NewHandler(checker *utils.HealthChecker) Handler {
	return &handler{checker: checker}
}

// @Summary Health check
// @Description Check the health status of the application
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} utils.HealthStatus
// @Failure 503 {object} utils.HealthStatus
// @Router /api/health [get]
func (h *handler) Check(c *gin.Context) {
	status := h.checker.Check(c.Request.Context())
	if status.Status == "healthy" {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}
