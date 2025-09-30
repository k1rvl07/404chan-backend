package message

import "time"

type Message struct {
	ID                 uint64    `json:"id" gorm:"primaryKey"`
	ThreadID           uint64    `json:"thread_id"`
	CreatedBySessionID uint64    `json:"created_by_session_id"`
	ParentID           *uint64   `json:"parent_id,omitempty"`
	Content            string    `json:"content"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	AuthorNickname     string    `json:"author_nickname"`
	IsAuthor           bool      `json:"is_author"`
}
