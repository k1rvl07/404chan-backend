package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"backend/internal/providers/redis"
)

type Service interface {
	CreateSessionAndUser(userAgent string, ipStr string) (*Session, *User, error)
	GetUserBySessionKey(sessionKey string) (*User, error)
	GetSessionByKey(sessionKey string) (*Session, error)
	UpdateSessionEndedAt(sessionID uint64) error
	GetSessionStartedAtBySessionKey(sessionKey string) (time.Time, error)
}

type service struct {
	repo   Repository
	redisP *redis.RedisProvider
}

func NewService(repo Repository, redisP *redis.RedisProvider) Service {
	return &service{repo: repo, redisP: redisP}
}

func (s *service) CreateSessionAndUser(userAgent, ipStr string) (*Session, *User, error) {
	user, err := s.repo.GetUserByIP(ipStr)
	if err != nil {
		user = &User{
			IP:       ipStr,
			Nickname: "Аноним",
		}
		if err := s.repo.CreateUser(user); err != nil {
			return nil, nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	if err := s.repo.CloseUserSessions(user.ID); err != nil {
		return nil, nil, err
	}

	sessionKey, err := generateSessionKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate session key: %w", err)
	}

	session := &Session{
		SessionKey: sessionKey,
		UserAgent:  &userAgent,
		UserID:     user.ID,
		StartedAt:  time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.repo.CreateSession(session); err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, user, nil
}

func (s *service) GetUserBySessionKey(sessionKey string) (*User, error) {
	session, err := s.repo.GetSessionByKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	user, err := s.repo.GetUserByID(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

func (s *service) GetSessionByKey(sessionKey string) (*Session, error) {
	return s.repo.GetSessionByKey(sessionKey)
}

func (s *service) UpdateSessionEndedAt(sessionID uint64) error {
	sessionData, err := s.repo.GetSessionByID(sessionID)
	if err == nil && sessionData != nil {
		cacheKey := fmt.Sprintf("user:%d:session:%d", sessionData.UserID, sessionData.ID)
		s.redisP.Client.Del(context.Background(), cacheKey)
	}

	return s.repo.UpdateSessionEndedAt(sessionID)
}

func (s *service) GetSessionStartedAtBySessionKey(sessionKey string) (time.Time, error) {
	session, err := s.repo.GetSessionByKey(sessionKey)
	if err != nil {
		return time.Time{}, err
	}
	return session.StartedAt, nil
}

func generateSessionKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
