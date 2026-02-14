package session

import "time"

type Session struct {
	ID         uint64    `gorm:"primaryKey"`
	SessionKey string    `gorm:"unique;not null"`
	StartedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	EndedAt    *time.Time
	UserAgent  *string   `gorm:"type:text"`
	UserID     uint64    `gorm:"not null;index"`
	CreatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

type User struct {
	ID        uint64    `gorm:"primaryKey"`
	IP        string    `gorm:"type:inet;not null;unique"`
	Nickname  string    `gorm:"not null;default:'Аноним'"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

type SessionResponse struct {
	ID         uint64    `json:"id"`
	Nickname   string    `json:"nickname"`
	CreatedAt  time.Time `json:"created_at"`
	SessionKey string    `json:"session_key"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
