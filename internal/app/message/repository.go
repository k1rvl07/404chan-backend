package message

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	CreateMessage(threadID uint64, sessionID uint64, parentID *uint64, content string, authorNickname string, isAuthor bool) (*Message, error)
	GetMessagesByThreadID(threadID uint64, page int, limit int) ([]*Message, int64, error)
	GetUserLastMessageTime(userID uint64) (*time.Time, error)
	GetMessageByID(id uint64) (*Message, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateMessage(
	threadID uint64,
	sessionID uint64,
	parentID *uint64,
	content string,
	authorNickname string,
	isAuthor bool,
) (*Message, error) {
	message := &Message{
		ThreadID:           threadID,
		CreatedBySessionID: sessionID,
		ParentID:           parentID,
		Content:            content,
		AuthorNickname:     authorNickname,
		IsAuthor:           isAuthor,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	result := r.db.Create(message)
	if result.Error != nil {
		return nil, result.Error
	}
	return message, nil
}

func (r *repository) GetMessagesByThreadID(threadID uint64, page int, limit int) ([]*Message, int64, error) {
	var messages []*Message
	var total int64
	offset := (page - 1) * limit

	err := r.db.Table("messages").
		Where("messages.thread_id = ?", threadID).
		Order("messages.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.Model(&Message{}).Where("thread_id = ?", threadID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

func (r *repository) GetUserLastMessageTime(userID uint64) (*time.Time, error) {
	var lastMessageTime sql.NullTime
	err := r.db.Model(&Message{}).
		Select("MAX(messages.created_at)").
		Joins("JOIN sessions ON sessions.id = messages.created_by_session_id").
		Where("sessions.user_id = ?", userID).
		Scan(&lastMessageTime).Error
	if err != nil {
		return nil, err
	}
	if !lastMessageTime.Valid {
		return nil, nil
	}
	return &lastMessageTime.Time, nil
}

func (r *repository) GetMessageByID(id uint64) (*Message, error) {
	var message Message
	err := r.db.Table("messages").
		Where("messages.id = ?", id).
		First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}
