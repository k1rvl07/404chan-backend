package message

import (
	"backend/internal/app/attachment"
	"backend/internal/app/session"
	"backend/internal/app/thread"
	"backend/internal/providers/minio"
	"backend/internal/providers/redis"
	"backend/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service interface {
	CreateMessage(ctx context.Context, threadID uint64, sessionKey string, content string, parentID *uint64, showAsAuthor bool, attachmentIDs []string) (*Message, error)
	GetMessagesByThreadID(ctx context.Context, threadID uint64, page int, limit int) ([]*Message, int64, error)
	GetUserLastMessageTime(userID uint64) (*time.Time, error)
	GetMessageCooldown(userID uint64) (*time.Time, error)
	GetMessageByID(ctx context.Context, id uint64) (*Message, error)
}

type service struct {
	repo          Repository
	sessionSvc    session.Service
	threadSvc     thread.Service
	dbConn        *gorm.DB
	redisP        *redis.RedisProvider
	minioP        *minio.MinioProvider
	eventBus      *utils.EventBus
	logger        *zap.SugaredLogger
	cachePrefix   string
	attachmentSvc attachment.Service
}

func NewService(
	repo Repository,
	sessionSvc session.Service,
	threadSvc thread.Service,
	dbConn *gorm.DB,
	redisP *redis.RedisProvider,
	eventBus *utils.EventBus,
	logger *zap.Logger,
	minioP *minio.MinioProvider,
	attachmentSvc attachment.Service,
) Service {
	return &service{
		repo:          repo,
		sessionSvc:    sessionSvc,
		threadSvc:     threadSvc,
		dbConn:        dbConn,
		redisP:        redisP,
		minioP:        minioP,
		eventBus:      eventBus,
		logger:        logger.Sugar(),
		cachePrefix:   "messages:thread",
		attachmentSvc: attachmentSvc,
	}
}

func (s *service) GetUserLastMessageTime(userID uint64) (*time.Time, error) {
	return s.repo.GetUserLastMessageTime(userID)
}

func (s *service) GetMessageCooldown(userID uint64) (*time.Time, error) {
	return s.GetUserLastMessageTime(userID)
}

func (s *service) CreateMessage(
	ctx context.Context,
	threadID uint64,
	sessionKey string,
	content string,
	parentID *uint64,
	showAsAuthor bool,
	attachmentIDs []string,
) (*Message, error) {
	contentLength := utf8.RuneCountInString(content)
	if contentLength < 1 || contentLength > 9999 {
		return nil, fmt.Errorf("message content must be between 1 and 9999 characters, got %d", contentLength)
	}

	user, err := s.sessionSvc.GetUserBySessionKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	lastMessageTime, err := s.GetUserLastMessageTime(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last message time: %w", err)
	}
	if lastMessageTime != nil {
		elapsed := time.Since(*lastMessageTime)
		if elapsed < 10*time.Second {
			secondsLeft := int64(10 - elapsed.Seconds())
			return nil, fmt.Errorf("message creation cooldown: %d seconds left", secondsLeft)
		}
	}

	session, err := s.sessionSvc.GetSessionByKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	thread, err := s.threadSvc.GetThreadByID(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	isThreadAuthor, err := s.threadSvc.IsUserAuthor(ctx, user.ID, threadID)
	if err != nil {
		s.logger.Warnw("Failed to check thread authorship", "error", err, "user_id", user.ID, "thread_id", threadID)
		isThreadAuthor = false
	}

	isAuthor := showAsAuthor && isThreadAuthor

	nickname := user.Nickname
	if nickname == "" {
		nickname = "Аноним"
	}

	message, err := s.repo.CreateMessage(threadID, session.ID, parentID, content, nickname, isAuthor)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	if len(attachmentIDs) > 0 && s.attachmentSvc != nil {
		if err := s.attachmentSvc.LinkToMessageByFileID(ctx, attachmentIDs, message.ID); err != nil {
			s.logger.Warn("Failed to link attachments to message", zap.Uint64("message_id", message.ID), zap.Error(err))
		}
	}

	s.dbConn.Exec(`
		INSERT INTO user_activity (user_id, message_count, created_at, updated_at)
		VALUES (?, 1, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			message_count = user_activity.message_count + 1,
			updated_at = NOW()
	`, user.ID)

	s.dbConn.Exec(`
		INSERT INTO threads_activity (thread_id, message_count, bump_at, created_at, updated_at)
		VALUES (?, 1, NOW(), NOW(), NOW())
		ON CONFLICT (thread_id) DO UPDATE SET
			message_count = threads_activity.message_count + 1,
			bump_at = NOW(),
			updated_at = NOW()
	`, threadID)

	s.invalidateCache(threadID)
	if s.threadSvc != nil {
		s.threadSvc.InvalidateThreadsCache(thread.BoardID)
		s.threadSvc.InvalidateTopThreadsCache()
	}

	userCacheKey := fmt.Sprintf("user:session:%s", sessionKey)
	s.redisP.Del(context.Background(), userCacheKey)

	eventData := map[string]interface{}{
		"message_id":      message.ID,
		"thread_id":       message.ThreadID,
		"content":         message.Content,
		"created_at":      message.CreatedAt,
		"updated_at":      message.UpdatedAt,
		"author_nickname": message.AuthorNickname,
		"is_author":       message.IsAuthor,
		"user_id":         user.ID,
		"timestamp":       time.Now().UTC().Unix(),
	}
	s.eventBus.Publish("message_created", eventData)

	return message, nil
}

func (s *service) GetMessagesByThreadID(
	ctx context.Context,
	threadID uint64,
	page int,
	limit int,
) ([]*Message, int64, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	cacheKey := fmt.Sprintf("%s:%d:page:%d:limit:%d", s.cachePrefix, threadID, page, limit)
	cmd := s.redisP.Get(ctx, cacheKey)
	cachedData, err := cmd.Result()
	var result struct {
		Messages []*Message `json:"messages"`
		Total    int64      `json:"total"`
	}

	if err == nil && cachedData != "" {
		if json.Unmarshal([]byte(cachedData), &result) == nil {
			return result.Messages, result.Total, nil
		}
	}

	messages, total, err := s.repo.GetMessagesByThreadID(threadID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get messages: %w", err)
	}

	if len(messages) > 0 && s.attachmentSvc != nil {
		for _, msg := range messages {
			attachments, err := s.attachmentSvc.GetByMessageID(ctx, msg.ID)
			if err == nil {
				msg.Attachments = make([]*MessageAttachment, 0, len(attachments))
				for _, att := range attachments {
					msg.Attachments = append(msg.Attachments, &MessageAttachment{
						ID:          att.FileID,
						FileID:      att.FileID,
						FileName:    att.FileName,
						FileURL:     att.FileURL,
						FileSize:    att.FileSize,
						ContentType: att.ContentType,
						ObjectName:  att.ObjectName,
						CreatedAt:   att.CreatedAt.Format("2006-01-02T15:04:05Z"),
					})
				}
			}
		}
	}

	if len(messages) > 0 {
		result.Messages = messages
		result.Total = total
		data, _ := json.Marshal(result)
		s.redisP.SetEX(ctx, cacheKey, data, 5*time.Minute)
	}

	return messages, total, nil
}

func (s *service) GetMessageByID(ctx context.Context, id uint64) (*Message, error) {
	cacheKey := fmt.Sprintf("%s:message:%d", s.cachePrefix, id)
	cmd := s.redisP.Get(ctx, cacheKey)
	cachedData, err := cmd.Result()

	if err == nil && cachedData != "" {
		var message Message
		if json.Unmarshal([]byte(cachedData), &message) == nil {
			return &message, nil
		}
	}

	message, err := s.repo.GetMessageByID(id)
	if err != nil {
		return nil, err
	}

	if message != nil && s.attachmentSvc != nil {
		attachments, err := s.attachmentSvc.GetByMessageID(ctx, message.ID)
		if err == nil {
			message.Attachments = make([]*MessageAttachment, 0, len(attachments))
			for _, att := range attachments {
				message.Attachments = append(message.Attachments, &MessageAttachment{
					ID:          att.FileID,
					FileID:      att.FileID,
					FileName:    att.FileName,
					FileURL:     att.FileURL,
					FileSize:    att.FileSize,
					ContentType: att.ContentType,
					ObjectName:  att.ObjectName,
					CreatedAt:   att.CreatedAt.Format("2006-01-02T15:04:05Z"),
				})
			}
		}
	}

	data, _ := json.Marshal(message)
	s.redisP.SetEX(ctx, cacheKey, data, 5*time.Minute)

	return message, nil
}

func (s *service) invalidateCache(threadID uint64) {
	ctx := context.Background()
	pattern := fmt.Sprintf("%s:%d:page:*", s.cachePrefix, threadID)
	var cursor uint64
	deletedCount := 0

	for {
		keys, cur, err := s.redisP.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			s.logger.Warnw("Redis scan failed during cache invalidation", "error", err, "pattern", pattern)
			return
		}

		if len(keys) > 0 {
			n, err := s.redisP.Del(ctx, keys...).Result()
			if err != nil {
				s.logger.Warnw("Failed to delete cache keys", "error", err, "keys", keys)
			} else {
				deletedCount += int(n)
			}
		}

		if cur == 0 {
			break
		}

		cursor = cur
	}

	if deletedCount > 0 {
		s.logger.Debugw("Message list cache invalidated", "thread_id", threadID, "deleted_keys", deletedCount)
	}
}
