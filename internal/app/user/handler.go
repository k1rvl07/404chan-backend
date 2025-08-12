package user

import (
	"context"
	"encoding/json"
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
}

func NewHandler(service Service, sessionSvc session.Service, eventBus *utils.EventBus, logger *zap.Logger, redisP *redis.RedisProvider) Handler {
	return &handler{
		service:    service,
		sessionSvc: sessionSvc,
		eventBus:   eventBus,
		logger:     logger.Sugar(),
		redisP:     redisP,
	}
}

func (h *handler) GetUser(c *gin.Context) {
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		h.logger.Warnw("GetUser: session_key missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_key is required"})
		return
	}

	sess, err := h.sessionSvc.GetSessionByKey(sessionKey)
	if err != nil {
		h.logger.Warnw("GetUser: session not found", "session_key", sessionKey)
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	cacheKey := fmt.Sprintf("user:%d:session:%d", sess.UserID, sess.ID)

	ctx := context.Background()
	if cached, err := h.redisP.Client.Get(ctx, cacheKey).Result(); err == nil {
		var data map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(cached), &data); jsonErr == nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}

	user, err := h.service.GetBySessionKey(sessionKey)
	if err != nil {
		h.logger.Warnw("GetUser: user not found", "session_key", sessionKey)
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	stats, err := h.service.GetStatsBySessionKey(sessionKey)
	if err != nil {
		stats = &UserActivity{UserID: user.ID, ThreadCount: 0, MessageCount: 0}
	}

	startedAt, err := h.sessionSvc.GetSessionStartedAtBySessionKey(sessionKey)
	if err != nil {
		h.logger.Warnw("GetUser: session not found", "session_key", sessionKey)
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	resp := gin.H{
		"ID":               user.ID,
		"Nickname":         user.Nickname,
		"CreatedAt":        user.CreatedAt,
		"SessionStartedAt": startedAt,
		"SessionKey":       sessionKey,
		"MessagesCount":    stats.MessageCount,
		"ThreadsCount":     stats.ThreadCount,
	}

	if dataBytes, err := json.Marshal(resp); err == nil {
		h.redisP.SetWithDefaultTTL(ctx, cacheKey, dataBytes, 0)
	}
	h.logger.Infow("GetUser: successful", "user_id", user.ID, "nickname", user.Nickname)
	c.JSON(http.StatusOK, resp)
}

func (h *handler) UpdateNickname(c *gin.Context) {
	var req struct {
		SessionKey string `json:"session_key" binding:"required"`
		Nickname   string `json:"nickname" binding:"required,min=1,max=16"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("UpdateNickname: invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ник должен быть 1-16 символов"})
		return
	}

	matched, err := regexp.MatchString(`^[\p{L}\p{N}]+$`, req.Nickname)
	if err != nil {
		h.logger.Errorw("UpdateNickname: regex failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate nickname"})
		return
	}
	if !matched {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ник должен содержать только буквы и цифры (без пробелов и символов)"})
		return
	}

	session, err := h.sessionSvc.GetSessionByKey(req.SessionKey)
	if err != nil {
		h.logger.Warnw("UpdateNickname: session not found", "session_key", req.SessionKey)
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if err := h.service.UpdateNickname(session.UserID, req.Nickname); err != nil {
		if err.Error() == "nickname can only be changed once per minute" {
			h.logger.Warnw("UpdateNickname: rate limited", "user_id", session.UserID)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Менять ник можно не чаще раза в минуту"})
			return
		}
		h.logger.Errorw("UpdateNickname: failed to update in DB", "user_id", session.UserID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update nickname"})
		return
	}

	cacheKey := fmt.Sprintf("user:%d:session:%d", session.UserID, session.ID)
	h.redisP.Client.Del(context.Background(), cacheKey)

	h.logger.Infow("UpdateNickname: DB updated", "user_id", session.UserID, "new_nickname", req.Nickname)
	eventData := map[string]interface{}{
		"user_id":   int(session.UserID),
		"nickname":  req.Nickname,
		"timestamp": time.Now().UTC().Unix(),
	}
	h.logger.Infow("UpdateNickname: publishing event", "event", "nickname_updated", "data", eventData)
	h.eventBus.Publish("nickname_updated", eventData)

	c.JSON(http.StatusOK, gin.H{
		"ID":                     session.UserID,
		"Nickname":               req.Nickname,
		"CreatedAt":              time.Now().UTC().Format(time.RFC3339),
		"SessionKey":             req.SessionKey,
		"MessagesCount":          0,
		"ThreadsCount":           0,
		"LastNicknameChangeUnix": time.Now().UTC().Unix(),
	})
}
