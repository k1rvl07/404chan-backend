package cleanup

import (
	"context"
	"time"

	"backend/internal/app/attachment"
	"backend/internal/app/message"
	"backend/internal/app/thread"
	"backend/internal/providers/minio"
	"backend/internal/providers/redis"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service interface {
	Cleanup(ctx context.Context, minutes int, cleanMessages, cleanThreads, cleanAttachments, cleanRedis bool) (CleanupResult, error)
}

type CleanupResult struct {
	MessagesDeleted    int64 `json:"messagesDeleted"`
	ThreadsDeleted     int64 `json:"threadsDeleted"`
	AttachmentsDeleted int64 `json:"attachmentsDeleted"`
	RedisFlushed       bool  `json:"redisFlushed"`
}

type service struct {
	db     *gorm.DB
	redisP *redis.RedisProvider
	minioP *minio.MinioProvider
	logger *zap.SugaredLogger
}

func NewService(db *gorm.DB, redisP *redis.RedisProvider, minioP *minio.MinioProvider, logger *zap.Logger) Service {
	return &service{
		db:     db,
		redisP: redisP,
		minioP: minioP,
		logger: logger.Sugar(),
	}
}

func (s *service) Cleanup(ctx context.Context, minutes int, cleanMessages, cleanThreads, cleanAttachments, cleanRedis bool) (CleanupResult, error) {
	result := CleanupResult{}

	cutoffDate := time.Now().Add(-time.Duration(minutes) * time.Minute)
	s.logger.Infow("Starting cleanup", "minutes", minutes, "cutoff", cutoffDate)

	if cleanMessages {
		var count int64
		s.db.Model(&message.Message{}).Where("created_at < ?", cutoffDate).Count(&count)
		res := s.db.Where("created_at < ?", cutoffDate).Delete(&message.Message{})
		result.MessagesDeleted = res.RowsAffected
		s.logger.Infow("Deleted messages", "count", result.MessagesDeleted)
	}

	if cleanThreads {
		var count int64
		s.db.Model(&thread.Thread{}).Where("created_at < ?", cutoffDate).Count(&count)
		res := s.db.Where("created_at < ?", cutoffDate).Delete(&thread.Thread{})
		result.ThreadsDeleted = res.RowsAffected
		s.logger.Infow("Deleted threads", "count", result.ThreadsDeleted)
	}

	if cleanAttachments {
		var attachments []attachment.Attachment
		s.db.Where("message_id IS NULL AND thread_id IS NULL").Find(&attachments)

		deleted := int64(0)
		for _, att := range attachments {
			if s.minioP != nil {
				err := s.minioP.DeleteFile(att.ObjectName)
				if err != nil {
					s.logger.Warnw("Failed to delete file from MinIO", "object", att.ObjectName)
					continue
				}
			}
			s.db.Delete(&att)
			deleted++
		}
		result.AttachmentsDeleted = deleted
		s.logger.Infow("Deleted orphaned attachments", "count", result.AttachmentsDeleted)
	}

	if cleanRedis {
		err := s.redisP.FlushDB(ctx)
		if err != nil {
			s.logger.Errorw("Failed to flush Redis", "error", err)
		} else {
			result.RedisFlushed = true
			s.logger.Infow("Flushed Redis")
		}
	}

	s.logger.Infow("Cleanup completed", "result", result)
	return result, nil
}
