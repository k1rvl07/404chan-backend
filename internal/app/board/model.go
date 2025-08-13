package board

import "time"

type Board struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	Slug        string    `json:"slug" gorm:"unique;not null"`
	Title       string    `json:"title" gorm:"not null"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
