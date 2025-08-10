package user

import (
	"fmt"
)

type Service interface {
	GetBySessionKey(sessionKey string) (*User, error)
	UpdateNickname(userID uint64, nickname string) error
	GetStatsBySessionKey(sessionKey string) (*UserActivity, error)
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
	return s.repo.UpdateUserNickname(userID, nickname)
}

func (s *service) GetStatsBySessionKey(sessionKey string) (*UserActivity, error) {
	session, err := s.repo.GetSessionByKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	return s.repo.GetUserActivityByUserID(session.UserID)
}
