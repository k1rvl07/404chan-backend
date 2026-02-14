package attachment

import (
	"context"
	"fmt"

	"backend/internal/providers/minio"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service interface {
	CreateTemporary(ctx context.Context, req *CreateAttachmentRequest) (*Attachment, error)
	LinkToThread(ctx context.Context, attachmentIDs []uint64, threadID uint64) error
	LinkToThreadByFileID(ctx context.Context, fileIDs []string, threadID uint64) error
	LinkToMessage(ctx context.Context, attachmentIDs []uint64, messageID uint64) error
	LinkToMessageByFileID(ctx context.Context, fileIDs []string, messageID uint64) error
	CreateThreadAttachments(ctx context.Context, threadID uint64, files []*UploadedFile) ([]*Attachment, error)
	CreateMessageAttachments(ctx context.Context, messageID uint64, files []*UploadedFile) ([]*Attachment, error)
	GetByThreadID(ctx context.Context, threadID uint64) ([]*Attachment, error)
	GetByMessageID(ctx context.Context, messageID uint64) ([]*Attachment, error)
	GetByIDs(ctx context.Context, ids []uint64) ([]*Attachment, error)
	GetByFileIDs(ctx context.Context, fileIDs []string) ([]*Attachment, error)
	GetTemporary(ctx context.Context) ([]*Attachment, error)
	UpdateObjectName(ctx context.Context, id uint64, objectName, fileURL string) error
	DeleteTemporary(ctx context.Context, fileID string) error
	DeleteByThreadID(ctx context.Context, threadID uint64) error
	DeleteByMessageID(ctx context.Context, messageID uint64) error
	DeleteAllByThreadID(ctx context.Context, threadID uint64) error
}

type service struct {
	repo   Repository
	db     *gorm.DB
	minioP *minio.MinioProvider
	logger *zap.Logger
}

func NewService(repo Repository, db *gorm.DB, minioP *minio.MinioProvider, logger *zap.Logger) Service {
	return &service{
		repo:   repo,
		db:     db,
		minioP: minioP,
		logger: logger,
	}
}

func (s *service) CreateTemporary(ctx context.Context, req *CreateAttachmentRequest) (*Attachment, error) {
	att := &Attachment{
		FileID:      req.FileID,
		FileName:    req.FileName,
		FileURL:     req.FileURL,
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
		ObjectName:  req.ObjectName,
	}

	if err := s.repo.Create(ctx, att); err != nil {
		s.logger.Error("Failed to create temporary attachment", zap.Error(err))
		return nil, fmt.Errorf("failed to create temporary attachment: %w", err)
	}

	s.logger.Info("Created temporary attachment",
		zap.Uint64("attachment_id", att.ID),
		zap.String("file_id", att.FileID),
	)

	return att, nil
}

func (s *service) LinkToThread(ctx context.Context, attachmentIDs []uint64, threadID uint64) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	threadIDPtr := &threadID
	return s.db.WithContext(ctx).
		Model(&Attachment{}).
		Where("id IN ?", attachmentIDs).
		Updates(map[string]interface{}{
			"thread_id":  threadIDPtr,
			"message_id": nil,
		}).Error
}

func (s *service) LinkToThreadByFileID(ctx context.Context, fileIDs []string, threadID uint64) error {
	if len(fileIDs) == 0 {
		return nil
	}

	threadIDPtr := &threadID
	return s.db.WithContext(ctx).
		Model(&Attachment{}).
		Where("file_id IN ?", fileIDs).
		Updates(map[string]interface{}{
			"thread_id":  threadIDPtr,
			"message_id": nil,
		}).Error
}

func (s *service) LinkToMessage(ctx context.Context, attachmentIDs []uint64, messageID uint64) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	messageIDPtr := &messageID
	return s.db.WithContext(ctx).
		Model(&Attachment{}).
		Where("id IN ?", attachmentIDs).
		Updates(map[string]interface{}{
			"thread_id":  nil,
			"message_id": messageIDPtr,
		}).Error
}

func (s *service) LinkToMessageByFileID(ctx context.Context, fileIDs []string, messageID uint64) error {
	if len(fileIDs) == 0 {
		return nil
	}

	messageIDPtr := &messageID
	return s.db.WithContext(ctx).
		Model(&Attachment{}).
		Where("file_id IN ?", fileIDs).
		Updates(map[string]interface{}{
			"thread_id":  nil,
			"message_id": messageIDPtr,
		}).Error
}

func (s *service) GetByIDs(ctx context.Context, ids []uint64) ([]*Attachment, error) {
	var attachments []*Attachment
	err := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&attachments).Error
	return attachments, err
}

func (s *service) GetByFileIDs(ctx context.Context, fileIDs []string) ([]*Attachment, error) {
	var attachments []*Attachment
	err := s.db.WithContext(ctx).
		Where("file_id IN ?", fileIDs).
		Find(&attachments).Error
	return attachments, err
}

func (s *service) UpdateObjectName(ctx context.Context, id uint64, objectName, fileURL string) error {
	return s.db.WithContext(ctx).
		Model(&Attachment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"object_name": objectName,
			"file_url":    fileURL,
		}).Error
}

func (s *service) GetTemporary(ctx context.Context) ([]*Attachment, error) {
	return s.repo.GetTemporary(ctx)
}

func (s *service) DeleteTemporary(ctx context.Context, fileID string) error {
	att, err := s.repo.GetByFileID(ctx, fileID)
	if err != nil {
		return err
	}

	if att.ObjectName != "" && s.minioP != nil {
		if err := s.minioP.DeleteFile(att.ObjectName); err != nil {
			s.logger.Warn("Failed to delete file from MinIO", zap.Error(err))
		}
	}

	return s.repo.DeleteByFileID(ctx, fileID)
}

func (s *service) CreateThreadAttachments(ctx context.Context, threadID uint64, files []*UploadedFile) ([]*Attachment, error) {
	if len(files) == 0 {
		return nil, nil
	}

	attachments := make([]*Attachment, 0, len(files))

	for _, file := range files {
		att := &Attachment{
			ThreadID:    &threadID,
			FileID:      file.ID,
			FileName:    file.Name,
			FileURL:     file.URL,
			FileSize:    file.Size,
			ContentType: file.ContentType,
			ObjectName:  file.ObjectName,
		}

		if err := s.repo.Create(ctx, att); err != nil {
			s.logger.Error("Failed to create attachment record", zap.Error(err))
			return nil, fmt.Errorf("failed to create attachment: %w", err)
		}

		attachments = append(attachments, att)
	}

	s.logger.Info("Created thread attachments",
		zap.Uint64("thread_id", threadID),
		zap.Int("count", len(attachments)),
	)

	return attachments, nil
}

func (s *service) CreateMessageAttachments(ctx context.Context, messageID uint64, files []*UploadedFile) ([]*Attachment, error) {
	if len(files) == 0 {
		return nil, nil
	}

	attachments := make([]*Attachment, 0, len(files))

	for _, file := range files {
		att := &Attachment{
			MessageID:   &messageID,
			FileID:      file.ID,
			FileName:    file.Name,
			FileURL:     file.URL,
			FileSize:    file.Size,
			ContentType: file.ContentType,
			ObjectName:  file.ObjectName,
		}

		if err := s.repo.Create(ctx, att); err != nil {
			s.logger.Error("Failed to create attachment record", zap.Error(err))
			return nil, fmt.Errorf("failed to create attachment: %w", err)
		}

		attachments = append(attachments, att)
	}

	s.logger.Info("Created message attachments",
		zap.Uint64("message_id", messageID),
		zap.Int("count", len(attachments)),
	)

	return attachments, nil
}

func (s *service) GetByThreadID(ctx context.Context, threadID uint64) ([]*Attachment, error) {
	return s.repo.GetByThreadID(ctx, threadID)
}

func (s *service) GetByMessageID(ctx context.Context, messageID uint64) ([]*Attachment, error) {
	return s.repo.GetByMessageID(ctx, messageID)
}

func (s *service) DeleteByThreadID(ctx context.Context, threadID uint64) error {
	attachments, err := s.repo.GetByThreadID(ctx, threadID)
	if err != nil {
		return err
	}

	objectNames := make([]string, 0, len(attachments))
	for _, att := range attachments {
		objectNames = append(objectNames, att.ObjectName)
	}

	if len(objectNames) > 0 && s.minioP != nil {
		if err := s.minioP.DeleteFiles(objectNames); err != nil {
			s.logger.Warn("Failed to delete files from MinIO", zap.Error(err))
		}
	}

	return s.repo.DeleteByThreadID(ctx, threadID)
}

func (s *service) DeleteByMessageID(ctx context.Context, messageID uint64) error {
	attachments, err := s.repo.GetByMessageID(ctx, messageID)
	if err != nil {
		return err
	}

	objectNames := make([]string, 0, len(attachments))
	for _, att := range attachments {
		objectNames = append(objectNames, att.ObjectName)
	}

	if len(objectNames) > 0 && s.minioP != nil {
		if err := s.minioP.DeleteFiles(objectNames); err != nil {
			s.logger.Warn("Failed to delete files from MinIO", zap.Error(err))
		}
	}

	return s.repo.DeleteByMessageID(ctx, messageID)
}

func (s *service) DeleteAllByThreadID(ctx context.Context, threadID uint64) error {
	return s.DeleteByThreadID(ctx, threadID)
}
