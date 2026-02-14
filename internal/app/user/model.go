package user

import "time"

type User struct {
	ID                   uint64     `gorm:"primaryKey"`
	IP                   string     `gorm:"type:inet;not null;unique"`
	Nickname             string     `gorm:"not null;default:'Аноним'"`
	LastNicknameChangeAt *time.Time `gorm:"column:last_nickname_change"`
	CreatedAt            time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt            time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

type UserActivity struct {
	UserID        uint64 `gorm:"primaryKey"`
	ThreadCount   int    `gorm:"not null;default:0"`
	MessageCount  int    `gorm:"not null;default:0"`
	LastMessageAt *time.Time
	LastThreadAt  *time.Time `gorm:"column:last_thread_at"`
	CreatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (UserActivity) TableName() string {
	return "user_activity"
}

type UpdateNicknameRequest struct {
	SessionKey string `json:"session_key" binding:"required"`
	Nickname   string `json:"nickname" binding:"required,min=1,max=16"`
}

type NicknameUpdateResponse struct {
	ID                     uint64    `json:"id"`
	Nickname               string    `json:"nickname"`
	CreatedAt              time.Time `json:"created_at"`
	SessionKey             string    `json:"session_key"`
	MessagesCount          int       `json:"messages_count"`
	ThreadsCount           int       `json:"threads_count"`
	LastNicknameChangeUnix int64     `json:"last_nickname_change_unix"`
}

type CooldownResponse struct {
	LastNicknameChangeUnix *int64 `json:"lastNicknameChangeUnix"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
