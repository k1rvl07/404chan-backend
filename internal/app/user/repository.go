package user

import (
	"backend/internal/app/session"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	GetSessionByKey(sessionKey string) (*session.Session, error)
	GetUserByID(id uint64) (*User, error)
	UpdateUserNickname(userID uint64, nickname string) error
	GetUserActivityByUserID(userID uint64) (*UserActivity, error)
	GetUserLastNicknameChange(userID uint64) (*time.Time, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetSessionByKey(sessionKey string) (*session.Session, error) {
	var session session.Session
	err := r.db.Where("session_key = ?", sessionKey).First(&session).Error
	return &session, err
}

func (r *repository) GetUserByID(id uint64) (*User, error) {
	var user User
	err := r.db.Where("id = ?", id).First(&user).Error
	return &user, err
}

func (r *repository) UpdateUserNickname(userID uint64, nickname string) error {
	return r.db.Model(&User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"nickname":             nickname,
			"last_nickname_change": time.Now().UTC(),
			"updated_at":           time.Now().UTC(),
		}).Error
}

func (r *repository) GetUserActivityByUserID(userID uint64) (*UserActivity, error) {
	var activity UserActivity
	err := r.db.Where("user_id = ?", userID).First(&activity).Error
	return &activity, err
}

func (r *repository) GetUserLastNicknameChange(userID uint64) (*time.Time, error) {
	var user User
	err := r.db.Select("last_nickname_change").Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return user.LastNicknameChangeAt, nil
}
