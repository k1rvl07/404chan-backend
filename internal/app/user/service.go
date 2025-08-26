package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend/internal/app/session"
	"backend/internal/providers/redis"

	"go.uber.org/zap"
)

const userCacheTTL = 5 * time.Minute

type UserResponse struct {
	ID               uint64    `json:"ID"`
	Nickname         string    `json:"Nickname"`
	CreatedAt        time.Time `json:"CreatedAt"`
	SessionStartedAt time.Time `json:"SessionStartedAt"`
	SessionKey       string    `json:"SessionKey"`
	MessagesCount    int       `json:"MessagesCount"`
	ThreadsCount     int       `json:"ThreadsCount"`
}

type Service interface {
	GetUserWithSession(ctx context.Context, sessionKey string) (*UserResponse, error)
	UpdateNickname(userID uint64, nickname string) error
	GetStatsBySessionKey(sessionKey string) (*UserActivity, error)
	GetUserLastThreadTime(userID uint64) (*time.Time, error)
	GetUserLastNicknameChange(userID uint64) (*time.Time, error)
}

type service struct {
	repo       Repository
	sessionSvc session.Service
	redisP     *redis.RedisProvider
	logger     *zap.SugaredLogger
}

func NewService(repo Repository, sessionSvc session.Service, redisP *redis.RedisProvider, logger *zap.Logger) Service {
	return &service{
		repo:       repo,
		sessionSvc: sessionSvc,
		redisP:     redisP,
		logger:     logger.Sugar(),
	}
}

func (s *service) GetUserWithSession(ctx context.Context, sessionKey string) (*UserResponse, error) {
	if sessionKey == "" {
		return nil, fmt.Errorf("session_key is required")
	}

	cacheKey := fmt.Sprintf("user:session:%s", sessionKey)

	cached, err := s.redisP.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var userResp UserResponse
		if json.Unmarshal([]byte(cached), &userResp) == nil {
			return &userResp, nil
		}
	}

	sess, err := s.sessionSvc.GetSessionByKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	user, err := s.repo.GetUserByID(sess.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	stats, err := s.repo.GetUserActivityByUserID(sess.UserID)
	if err != nil {
		stats = &UserActivity{UserID: user.ID, ThreadCount: 0, MessageCount: 0}
	}

	startedAt, err := s.sessionSvc.GetSessionStartedAtBySessionKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get session started at: %w", err)
	}

	userResp := &UserResponse{
		ID:               user.ID,
		Nickname:         user.Nickname,
		CreatedAt:        user.CreatedAt,
		SessionStartedAt: startedAt,
		SessionKey:       sessionKey,
		MessagesCount:    stats.MessageCount,
		ThreadsCount:     stats.ThreadCount,
	}

	data, err := json.Marshal(userResp)
	if err == nil {
		s.redisP.SetEX(ctx, cacheKey, data, userCacheTTL)
	}

	return userResp, nil
}

func (s *service) UpdateNickname(userID uint64, nickname string) error {
	lastChange, err := s.repo.GetUserLastNicknameChange(userID)
	if err != nil {
		return fmt.Errorf("failed to get last nickname change time: %w", err)
	}

	now := time.Now().UTC()
	if lastChange != nil && now.Sub(*lastChange) < time.Minute {
		return fmt.Errorf("nickname can only be changed once per minute")
	}

	return s.repo.UpdateUserNickname(userID, nickname)
}

func (s *service) GetStatsBySessionKey(sessionKey string) (*UserActivity, error) {
	session, err := s.repo.GetSessionByKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return s.repo.GetUserActivityByUserID(session.UserID)
}

func (s *service) GetUserLastThreadTime(userID uint64) (*time.Time, error) {
	return s.repo.GetUserLastThreadTime(userID)
}

func (s *service) GetUserLastNicknameChange(userID uint64) (*time.Time, error) {
	return s.repo.GetUserLastNicknameChange(userID)
}
