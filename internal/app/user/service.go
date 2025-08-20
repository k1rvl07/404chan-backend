package user

import (
	"fmt"
	"time"
)

type Service interface {
	GetBySessionKey(sessionKey string) (*User, error)
	UpdateNickname(userID uint64, nickname string) error
	GetStatsBySessionKey(sessionKey string) (*UserActivity, error)
	GetUserLastThreadTime(userID uint64) (*time.Time, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetBySessionKey(sessionKey string) (*User, error) {
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

func (s *service) UpdateNickname(userID uint64, nickname string) error {
	lastChange, err := s.repo.GetUserLastNicknameChange(userID)
	if err != nil {
		return fmt.Errorf("failed to get last nickname change time: %w", err)
	}

	now := time.Now().UTC()

	if lastChange == nil {
		return s.repo.UpdateUserNickname(userID, nickname)
	}

	if now.Sub(*lastChange) < time.Minute {
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
