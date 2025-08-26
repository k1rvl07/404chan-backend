package message

import (
	"net/http"
	"strconv"

	"backend/internal/app/session"
	"backend/internal/app/user"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	CreateMessage(c *gin.Context)
	GetMessagesByThreadID(c *gin.Context)
	GetMessageCooldown(c *gin.Context)
	GetMessageByID(c *gin.Context)
}

type handler struct {
	service    Service
	sessionSvc session.Service
	userSvc    user.Service
}

func NewHandler(service Service, sessionSvc session.Service, userSvc user.Service) Handler {
	return &handler{
		service:    service,
		sessionSvc: sessionSvc,
		userSvc:    userSvc,
	}
}

func (h *handler) CreateMessage(c *gin.Context) {
	threadIDStr := c.Param("thread_id")
	threadID, err := strconv.ParseUint(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread ID"})
		return
	}

	var req struct {
		Content  string  `json:"content" binding:"required"`
		ParentID *uint64 `json:"parent_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session_key is required"})
		return
	}

	message, err := h.service.CreateMessage(c.Request.Context(), threadID, sessionKey, req.Content, req.ParentID)
	if err != nil {
		if err.Error() == "message creation cooldown: ..." {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *handler) GetMessagesByThreadID(c *gin.Context) {
	threadIDStr := c.Param("thread_id")
	threadID, err := strconv.ParseUint(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread ID"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	messages, total, err := h.service.GetMessagesByThreadID(c.Request.Context(), threadID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get messages"})
		return
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

func (h *handler) GetMessageCooldown(c *gin.Context) {
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session_key is required"})
		return
	}

	user, err := h.sessionSvc.GetUserBySessionKey(sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	lastMessageTime, err := h.service.GetMessageCooldown(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get last message time"})
		return
	}

	var lastMessageUnix *int64
	if lastMessageTime != nil {
		unixTime := lastMessageTime.Unix()
		lastMessageUnix = &unixTime
	}

	c.JSON(http.StatusOK, gin.H{
		"lastMessageCreationUnix": lastMessageUnix,
	})
}

func (h *handler) GetMessageByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	message, err := h.service.GetMessageByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	c.JSON(http.StatusOK, message)
}
