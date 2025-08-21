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
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}
