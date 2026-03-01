package cleanup

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	Cleanup(c *gin.Context)
}

type handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return &handler{service: service}
}

type CleanupRequest struct {
	Minutes          int  `json:"minutes"`
	CleanMessages    bool `json:"cleanMessages"`
	CleanThreads     bool `json:"cleanThreads"`
	CleanAttachments bool `json:"cleanAttachments"`
	CleanRedis       bool `json:"cleanRedis"`
}

// @Summary Cleanup old data
// @Description Delete messages, threads, attachments and Redis cache older than specified minutes
// @Tags Cleanup
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param minutes query int false "Minutes (default: 1440)"
// @Param messages query bool false "Clean old messages"
// @Param threads query bool false "Clean old threads"
// @Param attachments query bool false "Clean old attachments"
// @Param redis query bool false "Clean Redis cache"
// @Success 200 {object} CleanupResult
// @Router /cleanup [post]
func (h *handler) Cleanup(c *gin.Context) {
	minutesStr := c.DefaultQuery("minutes", "1440")
	minutes, err := strconv.Atoi(minutesStr)
	if err != nil || minutes < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid minutes parameter"})
		return
	}

	cleanMessages := c.Query("messages") == "true"
	cleanThreads := c.Query("threads") == "true"
	cleanAttachments := c.Query("attachments") == "true"
	cleanRedis := c.Query("redis") == "true"

	cleanAll := !cleanMessages && !cleanThreads && !cleanAttachments && !cleanRedis

	result, err := h.service.Cleanup(c.Request.Context(), minutes, cleanAll || cleanMessages, cleanAll || cleanThreads, cleanAll || cleanAttachments, cleanAll || cleanRedis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
