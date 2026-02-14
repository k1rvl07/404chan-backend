package attachment

import "time"

type Attachment struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	ThreadID    *uint64   `json:"thread_id,omitempty" gorm:"index"`
	MessageID   *uint64   `json:"message_id,omitempty" gorm:"index"`
	FileID      string    `json:"file_id" gorm:"type:varchar(36);not null"`
	FileName    string    `json:"file_name" gorm:"not null"`
	FileURL     string    `json:"file_url" gorm:"not null"`
	FileSize    int64     `json:"file_size" gorm:"not null"`
	ContentType string    `json:"content_type" gorm:"type:varchar(100);not null"`
	ObjectName  string    `json:"object_name" gorm:"type:varchar(500);not null"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Attachment) TableName() string {
	return "attachments"
}

type UploadedFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ObjectName  string `json:"object_name"`
}

type CreateAttachmentRequest struct {
	ThreadID    *uint64 `json:"thread_id" binding:"required"`
	MessageID   *uint64 `json:"message_id"`
	FileID      string  `json:"file_id" binding:"required"`
	FileName    string  `json:"file_name" binding:"required"`
	FileURL     string  `json:"file_url" binding:"required"`
	FileSize    int64   `json:"file_size" binding:"required"`
	ContentType string  `json:"content_type" binding:"required"`
	ObjectName  string  `json:"object_name" binding:"required"`
}

type AttachmentListResponse struct {
	Attachments []*Attachment `json:"attachments"`
}

type DeleteTemporaryResponse struct {
	Success bool `json:"success"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
