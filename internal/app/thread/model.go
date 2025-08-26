package thread

import "time"

type Thread struct {
	ID             uint64    `json:"id"`
	BoardID        uint64    `json:"board_id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedBy      uint64    `json:"created_by"`
	AuthorNickname string    `json:"author_nickname"`
	MessagesCount  int       `json:"messages_count"`
}
