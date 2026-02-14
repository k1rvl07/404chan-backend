package message

import (
	"backend/internal/app/session"
	"net/http"
	"strconv"

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
}

func NewHandler(service Service, sessionSvc session.Service) Handler {
	return &handler{
		service:    service,
		sessionSvc: sessionSvc,
	}
}

// @Summary Create a new message
// @Description Create a new message in a thread
// @Tags Message
// @Accept json
// @Produce json
// @Param thread_id path int true "Thread ID"
// @Param request body CreateMessageRequest true "Message creation request"
// @Success 201 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/messages/{thread_id} [post]
func (h *handler) CreateMessage(c *gin.Context) {
	threadIDStr := c.Param("thread_id")
	threadID, err := strconv.ParseUint(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid thread ID"})
		return
	}
	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "session_key is required"})
		return
	}
	message, err := h.service.CreateMessage(
		c.Request.Context(),
		threadID,
		sessionKey,
		req.Content,
		req.ParentID,
		req.ShowAsAuthor,
		req.AttachmentIDs,
	)
	if err != nil {
		if err.Error() == "message creation cooldown: ..." {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, message)
}

// @Summary Get messages by thread ID
// @Description Get paginated list of messages for a thread
// @Tags Message
// @Accept json
// @Produce json
// @Param thread_id path int true "Thread ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} MessageListResponse
// @Router /api/messages/{thread_id} [get]
func (h *handler) GetMessagesByThreadID(c *gin.Context) {
	threadIDStr := c.Param("thread_id")
	threadID, err := strconv.ParseUint(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid thread ID"})
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to get messages"})
		return
	}
	totalPages := (total + int64(limit) - 1) / int64(limit)
	c.JSON(http.StatusOK, MessageListResponse{
		Messages: messages,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// @Summary Get message creation cooldown
// @Description Get the timestamp of the last message creation
// @Tags Message
// @Accept json
// @Produce json
// @Param session_key query string true "Session key"
// @Success 200 {object} MessageCooldownResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/messages/cooldown [get]
func (h *handler) GetMessageCooldown(c *gin.Context) {
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "session_key is required"})
		return
	}
	user, err := h.sessionSvc.GetUserBySessionKey(sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	lastMessageTime, err := h.service.GetMessageCooldown(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to get last message time"})
		return
	}
	var lastMessageUnix *int64
	if lastMessageTime != nil {
		unixTime := lastMessageTime.Unix()
		lastMessageUnix = &unixTime
	}
	c.JSON(http.StatusOK, MessageCooldownResponse{
		LastMessageCreationUnix: lastMessageUnix,
	})
}

// @Summary Get message by ID
// @Description Get a specific message by its ID
// @Tags Message
// @Accept json
// @Produce json
// @Param id path int true "Message ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/messages/message/{id} [get]
func (h *handler) GetMessageByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid message ID"})
		return
	}
	message, err := h.service.GetMessageByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "message not found"})
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: message})
}
