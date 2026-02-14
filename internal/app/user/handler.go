package user

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"backend/internal/app/session"
	"backend/internal/providers/redis"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type handler struct {
	service    Service
	sessionSvc session.Service
	eventBus   *utils.EventBus
	logger     *zap.SugaredLogger
	redisP     *redis.RedisProvider
}

type Handler interface {
	GetUser(c *gin.Context)
	UpdateNickname(c *gin.Context)
	GetCooldown(c *gin.Context)
}

func NewHandler(
	service Service,
	sessionSvc session.Service,
	eventBus *utils.EventBus,
	logger *zap.Logger,
	redisP *redis.RedisProvider,
) Handler {
	return &handler{
		service:    service,
		sessionSvc: sessionSvc,
		eventBus:   eventBus,
		logger:     logger.Sugar(),
		redisP:     redisP,
	}
}

// @Summary Get user profile
// @Description Get user profile by session key
// @Tags User
// @Accept json
// @Produce json
// @Param session_key query string true "Session key"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/user [get]
func (h *handler) GetUser(c *gin.Context) {
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		h.logger.Warnw("GetUser: session_key missing")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "session_key is required"})
		return
	}

	ctx := c.Request.Context()
	userResp, err := h.service.GetUserWithSession(ctx, sessionKey)
	if err != nil {
		h.logger.Warnw("GetUser: failed to get user", "session_key", sessionKey, "error", err)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	h.logger.Infow("GetUser: successful", "user_id", userResp.ID, "nickname", userResp.Nickname)
	c.JSON(http.StatusOK, userResp)
}

// @Summary Update user nickname
// @Description Update user's nickname (1-16 alphanumeric characters)
// @Tags User
// @Accept json
// @Produce json
// @Param request body UpdateNicknameRequest true "Nickname update request"
// @Success 200 {object} NicknameUpdateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/user/nickname [patch]
func (h *handler) UpdateNickname(c *gin.Context) {
	var req UpdateNicknameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("UpdateNickname: invalid request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Ник должен быть 1-16 символов"})
		return
	}

	matched, err := regexp.MatchString(`^[\p{L}\p{N}]+$`, req.Nickname)
	if err != nil {
		h.logger.Errorw("UpdateNickname: regex failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to validate nickname"})
		return
	}
	if !matched {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Ник должен содержать только буквы и цифры (без пробелов и символов)"})
		return
	}

	session, err := h.sessionSvc.GetSessionByKey(req.SessionKey)
	if err != nil {
		h.logger.Warnw("UpdateNickname: session not found", "session_key", req.SessionKey)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "session not found"})
		return
	}

	if err := h.service.UpdateNickname(session.UserID, req.Nickname); err != nil {
		if err.Error() == "nickname can only be changed once per minute" {
			h.logger.Warnw("UpdateNickname: rate limited", "user_id", session.UserID)
			c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: "Менять ник можно не чаще раза в минуту"})
			return
		}
		h.logger.Errorw("UpdateNickname: failed to update in DB", "user_id", session.UserID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update nickname"})
		return
	}

	cacheKey := fmt.Sprintf("user:session:%s", req.SessionKey)
	h.redisP.Del(context.Background(), cacheKey)

	h.logger.Infow("UpdateNickname: DB updated", "user_id", session.UserID, "new_nickname", req.Nickname)
	eventData := map[string]interface{}{
		"user_id":   int(session.UserID),
		"nickname":  req.Nickname,
		"timestamp": time.Now().UTC().Unix(),
	}
	h.logger.Infow("UpdateNickname: publishing event", "event", "nickname_updated", "data", eventData)
	h.eventBus.Publish("nickname_updated", eventData)

	c.JSON(http.StatusOK, NicknameUpdateResponse{
		ID:                     session.UserID,
		Nickname:               req.Nickname,
		CreatedAt:              time.Now().UTC(),
		SessionKey:             req.SessionKey,
		MessagesCount:          0,
		ThreadsCount:           0,
		LastNicknameChangeUnix: time.Now().UTC().Unix(),
	})
}

// @Summary Get nickname change cooldown
// @Description Get the timestamp of the last nickname change
// @Tags User
// @Accept json
// @Produce json
// @Param session_key query string true "Session key"
// @Success 200 {object} CooldownResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/user/cooldown [get]
func (h *handler) GetCooldown(c *gin.Context) {
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		h.logger.Warnw("GetCooldown: session_key missing")
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "session_key is required"})
		return
	}

	session, err := h.sessionSvc.GetSessionByKey(sessionKey)
	if err != nil {
		h.logger.Warnw("GetCooldown: session not found", "session_key", sessionKey)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "session not found"})
		return
	}

	lastChange, err := h.service.GetUserLastNicknameChange(session.UserID)
	if err != nil {
		h.logger.Errorw("GetCooldown: failed to get last nickname change", "user_id", session.UserID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to get last nickname change"})
		return
	}

	var lastChangeUnix *int64
	if lastChange != nil {
		unixTime := lastChange.Unix()
		lastChangeUnix = &unixTime
	}

	c.JSON(http.StatusOK, CooldownResponse{
		LastNicknameChangeUnix: lastChangeUnix,
	})
}
