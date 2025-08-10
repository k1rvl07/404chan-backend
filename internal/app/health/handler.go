package health

import (
	"net/http"

	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type HealthController struct {
	checker *utils.HealthChecker
}

func NewHealthController(checker *utils.HealthChecker) *HealthController {
	return &HealthController{checker: checker}
}

func (h *HealthController) Check(c *gin.Context) {
	status := h.checker.Check(c.Request.Context())
	if status.Status == "healthy" {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}
