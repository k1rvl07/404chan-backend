package session

import (
	"gorm.io/gorm"
	"time"
)

type Repository interface {
	GetUserByIP(ip string) (*User, error)
	CreateUser(user *User) error
	CreateSession(session *Session) error
	GetSessionByKey(sessionKey string) (*Session, error)
	GetUserByID(id uint64) (*User, error)
	UpdateSessionEndedAt(sessionID uint64) error
	CloseUserSessions(userID uint64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetUserByIP(ip string) (*User, error) {
	var user User
	err := r.db.Where("ip = ?", ip).First(&user).Error
	return &user, err
}

func (r *repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

func (r *repository) CreateSession(session *Session) error {
	return r.db.Create(session).Error
}

func (r *repository) GetSessionByKey(sessionKey string) (*Session, error) {
	var session Session
	err := r.db.Where("session_key = ?", sessionKey).First(&session).Error
	return &session, err
}

func (r *repository) GetUserByID(id uint64) (*User, error) {
	var user User
	err := r.db.Where("id = ?", id).First(&user).Error
	return &user, err
}

func (r *repository) UpdateSessionEndedAt(sessionID uint64) error {
	return r.db.Model(&Session{}).
		Where("id = ?", sessionID).
		Update("ended_at", time.Now().UTC()).Error
}

func (r *repository) CloseUserSessions(userID uint64) error {
	return r.db.Model(&Session{}).
		Where("user_id = ? AND ended_at IS NULL", userID).
		Update("ended_at", time.Now().UTC()).Error
}
