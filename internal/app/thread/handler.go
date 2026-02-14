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

// @Summary Create a new thread
// @Description Create a new thread in a board
// @Tags Thread
// @Accept json
// @Produce json
// @Param board_id path int true "Board ID"
// @Param request body CreateThreadRequest true "Thread creation request"
// @Success 201 {object} ThreadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/threads/{board_id} [post]
func (h *handler) CreateThread(c *gin.Context) {
	boardIDStr := c.Param("board_id")
	boardID, err := strconv.ParseUint(boardIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid board ID"})
		return
	}

	var req CreateThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "session_key is required"})
		return
	}

	thread, err := h.service.CreateThread(c.Request.Context(), boardID, sessionKey, req.Title, req.Content, req.AttachmentIDs)
	if err != nil {
		if err.Error() == "thread creation cooldown: ..." {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, thread)
}

// @Summary Get threads by board ID
// @Description Get paginated list of threads for a board
// @Tags Thread
// @Accept json
// @Produce json
// @Param board_id path int true "Board ID"
// @Param sort query string false "Sort order (new, top)" default("new")
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} ThreadListResponse
// @Router /api/threads/{board_id} [get]
func (h *handler) GetThreadsByBoardID(c *gin.Context) {
	boardIDStr := c.Param("board_id")
	boardID, err := strconv.ParseUint(boardIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid board ID"})
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to get threads"})
		return
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, ThreadListResponse{
		Threads: threads,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// @Summary Get thread creation cooldown
// @Description Get the timestamp of the last thread creation
// @Tags Thread
// @Accept json
// @Produce json
// @Param session_key query string true "Session key"
// @Success 200 {object} ThreadCooldownResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/threads/cooldown [get]
func (h *handler) GetThreadCooldown(c *gin.Context) {
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

	lastThreadTime, err := h.userSvc.GetUserLastThreadTime(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to get last thread time"})
		return
	}

	var lastThreadUnix *int64
	if lastThreadTime != nil {
		unixTime := lastThreadTime.Unix()
		lastThreadUnix = &unixTime
	}

	c.JSON(http.StatusOK, ThreadCooldownResponse{
		LastThreadCreationUnix: lastThreadUnix,
	})
}

// @Summary Get thread by ID
// @Description Get a specific thread by its ID
// @Tags Thread
// @Accept json
// @Produce json
// @Param id path int true "Thread ID"
// @Success 200 {object} ThreadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/threads/thread/{id} [get]
func (h *handler) GetThreadByID(c *gin.Context) {
	threadIDStr := c.Param("id")
	threadID, err := strconv.ParseUint(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid thread ID"})
		return
	}

	thread, err := h.service.GetThreadByID(c.Request.Context(), threadID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "thread not found"})
		return
	}

	c.JSON(http.StatusOK, thread)
}

// @Summary Get top threads
// @Description Get paginated list of top threads across all boards
// @Tags Thread
// @Accept json
// @Produce json
// @Param sort query string false "Sort order (new, top)" default("new")
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} TopThreadsResponse
// @Router /api/threads/top [get]
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to get top threads"})
		return
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, TopThreadsResponse{
		Threads: threads,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// @Summary Check thread authorship
// @Description Check if the current user is the author of a thread
// @Tags Thread
// @Accept json
// @Produce json
// @Param thread_id path int true "Thread ID"
// @Param session_key query string true "Session key"
// @Success 200 {object} CheckAuthorResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/threads/check-author/{thread_id} [get]
func (h *handler) CheckThreadAuthor(c *gin.Context) {
	threadIDStr := c.Param("thread_id")
	threadID, err := strconv.ParseUint(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid thread ID"})
		return
	}

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

	isAuthor, err := h.service.IsUserAuthor(c.Request.Context(), user.ID, threadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to check authorship"})
		return
	}

	c.JSON(http.StatusOK, CheckAuthorResponse{IsAuthor: isAuthor})
}
