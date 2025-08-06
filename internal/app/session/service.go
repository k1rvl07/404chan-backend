package session

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "time"
)

type Service interface {
    CreateSessionAndUser(userAgent string, ipStr string) (*Session, *User, error)
    GetUserBySessionKey(sessionKey string) (*User, error)
    GetSessionByKey(sessionKey string) (*Session, error)
    UpdateSessionEndedAt(sessionID uint64) error
}

type service struct {
    repo Repository
}

func NewService(repo Repository) Service {
    return &service{repo: repo}
}

func (s *service) CreateSessionAndUser(userAgent string, ipStr string) (*Session, *User, error) {
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

    _ = s.repo.CloseUserSessions(user.ID)

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
    return s.repo.UpdateSessionEndedAt(sessionID)
}

func generateSessionKey() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}