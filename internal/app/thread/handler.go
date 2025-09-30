package thread

import (
	"net/http"
	"strconv"

	"backend/internal/app/session"
	"backend/internal/app/user"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	CreateThread(c *gin.Context)
	GetThreadsByBoardID(c *gin.Context)
	GetThreadCooldown(c *gin.Context)
	GetThreadByID(c *gin.Context)
	GetTopThreads(c *gin.Context)
	CheckThreadAuthor(c *gin.Context)
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

func (h *handler) CreateThread(c *gin.Context) {
	boardIDStr := c.Param("board_id")
	boardID, err := strconv.ParseUint(boardIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
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

	thread, err := h.service.CreateThread(c.Request.Context(), boardID, sessionKey, req.Title, req.Content)
	if err != nil {
		if err.Error() == "thread creation cooldown: ..." {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, thread)
}

func (h *handler) GetThreadsByBoardID(c *gin.Context) {
	boardIDStr := c.Param("board_id")
	boardID, err := strconv.ParseUint(boardIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	sort := c.DefaultQuery("sort", "new")
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

	threads, total, err := h.service.GetThreadsByBoardID(c.Request.Context(), boardID, sort, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get threads"})
		return
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, gin.H{
		"threads": threads,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

func (h *handler) GetThreadCooldown(c *gin.Context) {
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

	lastThreadTime, err := h.userSvc.GetUserLastThreadTime(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get last thread time"})
		return
	}

	var lastThreadUnix *int64
	if lastThreadTime != nil {
		unixTime := lastThreadTime.Unix()
		lastThreadUnix = &unixTime
	}

	c.JSON(http.StatusOK, gin.H{
		"lastThreadCreationUnix": lastThreadUnix,
	})
}

func (h *handler) GetThreadByID(c *gin.Context) {
	threadIDStr := c.Param("id")
	threadID, err := strconv.ParseUint(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread ID"})
		return
	}

	thread, err := h.service.GetThreadByID(c.Request.Context(), threadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "thread not found"})
		return
	}

	c.JSON(http.StatusOK, thread)
}

func (h *handler) GetTopThreads(c *gin.Context) {
	sort := c.DefaultQuery("sort", "new")
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

	threads, total, err := h.service.GetTopThreads(c.Request.Context(), sort, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get top threads"})
		return
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, gin.H{
		"threads": threads,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

func (h *handler) CheckThreadAuthor(c *gin.Context) {
	threadIDStr := c.Param("thread_id")
	threadID, err := strconv.ParseUint(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread ID"})
		return
	}

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

	isAuthor, err := h.service.IsUserAuthor(c.Request.Context(), user.ID, threadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check authorship"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_author": isAuthor})
}
