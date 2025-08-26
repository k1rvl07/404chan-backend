package message

import "time"

type Message struct {
	ID             uint64    `json:"id" gorm:"primaryKey"`
	ThreadID       uint64    `json:"thread_id"`
	UserID         uint64    `json:"user_id"`
	ParentID       *uint64   `json:"parent_id,omitempty"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	AuthorNickname string    `json:"author_nickname"`
}
