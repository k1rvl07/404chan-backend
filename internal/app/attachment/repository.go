package attachment

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, att *Attachment) error
	GetByThreadID(ctx context.Context, threadID uint64) ([]*Attachment, error)
	GetByMessageID(ctx context.Context, messageID uint64) ([]*Attachment, error)
	GetByFileID(ctx context.Context, fileID string) (*Attachment, error)
	GetTemporary(ctx context.Context) ([]*Attachment, error)
	Delete(ctx context.Context, id uint64) error
	DeleteByFileID(ctx context.Context, fileID string) error
	DeleteByThreadID(ctx context.Context, threadID uint64) error
	DeleteByMessageID(ctx context.Context, messageID uint64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, att *Attachment) error {
	return r.db.WithContext(ctx).Create(att).Error
}

func (r *repository) GetByThreadID(ctx context.Context, threadID uint64) ([]*Attachment, error) {
	var attachments []*Attachment
	err := r.db.WithContext(ctx).
		Where("thread_id = ?", threadID).
		Order("created_at ASC").
		Find(&attachments).Error
	return attachments, err
}

func (r *repository) GetByMessageID(ctx context.Context, messageID uint64) ([]*Attachment, error) {
	var attachments []*Attachment
	err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Order("created_at ASC").
		Find(&attachments).Error
	return attachments, err
}

func (r *repository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&Attachment{}, id).Error
}

func (r *repository) GetByFileID(ctx context.Context, fileID string) (*Attachment, error) {
	var attachment Attachment
	err := r.db.WithContext(ctx).Where("file_id = ?", fileID).First(&attachment).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

func (r *repository) GetTemporary(ctx context.Context) ([]*Attachment, error) {
	var attachments []*Attachment
	err := r.db.WithContext(ctx).
		Where("thread_id IS NULL AND message_id IS NULL").
		Order("created_at ASC").
		Find(&attachments).Error
	return attachments, err
}

func (r *repository) DeleteByFileID(ctx context.Context, fileID string) error {
	return r.db.WithContext(ctx).Where("file_id = ?", fileID).Delete(&Attachment{}).Error
}

func (r *repository) DeleteByThreadID(ctx context.Context, threadID uint64) error {
	return r.db.WithContext(ctx).
		Where("thread_id = ?", threadID).
		Delete(&Attachment{}).Error
}

func (r *repository) DeleteByMessageID(ctx context.Context, messageID uint64) error {
	return r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Delete(&Attachment{}).Error
}
