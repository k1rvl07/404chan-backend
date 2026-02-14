package message

import "time"

type Message struct {
	ID                 uint64               `json:"id" gorm:"primaryKey"`
	ThreadID           uint64               `json:"thread_id"`
	CreatedBySessionID uint64               `json:"created_by_session_id"`
	ParentID           *uint64              `json:"parent_id,omitempty"`
	Content            string               `json:"content"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	AuthorNickname     string               `json:"author_nickname"`
	IsAuthor           bool                 `json:"is_author"`
	Attachments        []*MessageAttachment `json:"attachments,omitempty" gorm:"-"`
}

type MessageAttachment struct {
	ID          string `json:"id"`
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name"`
	FileURL     string `json:"file_url"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type"`
	ObjectName  string `json:"object_name"`
	CreatedAt   string `json:"created_at"`
}

type CreateMessageRequest struct {
	Content       string   `json:"content" binding:"required"`
	ParentID      *uint64  `json:"parent_id,omitempty"`
	ShowAsAuthor  bool     `json:"show_as_author"`
	AttachmentIDs []string `json:"attachment_ids"`
}

type MessageListResponse struct {
	Messages   []*Message `json:"messages"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

type MessageResponse struct {
	Message *Message `json:"message"`
}

type MessageCooldownResponse struct {
	LastMessageCreationUnix *int64 `json:"lastMessageCreationUnix"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
