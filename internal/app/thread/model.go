package thread

import "time"

type Thread struct {
	ID                 uint64              `json:"id" gorm:"primaryKey"`
	BoardID            uint64              `json:"board_id"`
	BoardSlug          string              `json:"board_slug"`
	Title              string              `json:"title"`
	Content            string              `json:"content"`
	CreatedBySessionID uint64              `json:"created_by_session_id"`
	AuthorNickname     string              `json:"author_nickname"`
	MessagesCount      int                 `json:"messages_count"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	Attachments        []*ThreadAttachment `json:"attachments,omitempty" gorm:"-"`
}

type ThreadAttachment struct {
	ID          string `json:"id"`
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name"`
	FileURL     string `json:"file_url"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type"`
	ObjectName  string `json:"object_name"`
	CreatedAt   string `json:"created_at"`
}

type ThreadActivity struct {
	ThreadID     uint64    `json:"thread_id" gorm:"primaryKey;column:thread_id"`
	MessageCount int       `json:"message_count" gorm:"not null;default:0"`
	BumpAt       time.Time `json:"bump_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (ThreadActivity) TableName() string {
	return "threads_activity"
}

type CreateThreadRequest struct {
	Title         string   `json:"title" binding:"required"`
	Content       string   `json:"content" binding:"required"`
	AttachmentIDs []string `json:"attachment_ids"`
}

type ThreadListResponse struct {
	Threads    []*Thread  `json:"threads"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

type ThreadResponse struct {
	Thread *Thread `json:"thread"`
}

type TopThreadsResponse struct {
	Threads    []*Thread  `json:"threads"`
	Pagination Pagination `json:"pagination"`
}

type ThreadCooldownResponse struct {
	LastThreadCreationUnix *int64 `json:"lastThreadCreationUnix"`
}

type CheckAuthorResponse struct {
	IsAuthor bool `json:"is_author"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
