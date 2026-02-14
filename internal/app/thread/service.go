package thread

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"backend/internal/app/attachment"
	"backend/internal/app/session"
	"backend/internal/app/user"
	"backend/internal/providers/minio"
	"backend/internal/providers/redis"
	"backend/internal/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service interface {
	CreateThread(ctx context.Context, boardID uint64, sessionKey, title, content string, attachmentIDs []string) (*Thread, error)
	GetThreadsByBoardID(ctx context.Context, boardID uint64, sort string, page, limit int) ([]*Thread, int64, error)
	GetThreadByID(ctx context.Context, threadID uint64) (*Thread, error)
	GetUserLastThreadTime(userID uint64) (*time.Time, error)
	InvalidateThreadsCache(boardID uint64)
	GetTopThreads(ctx context.Context, sort string, page, limit int) ([]*Thread, int64, error)
	InvalidateTopThreadsCache()
	IsUserAuthor(ctx context.Context, userID uint64, threadID uint64) (bool, error)
}

type service struct {
	repo          Repository
	sessionSvc    session.Service
	userSvc       user.Service
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
	userSvc user.Service,
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
		userSvc:       userSvc,
		dbConn:        dbConn,
		redisP:        redisP,
		minioP:        minioP,
		eventBus:      eventBus,
		logger:        logger.Sugar(),
		cachePrefix:   "threads:board",
		attachmentSvc: attachmentSvc,
	}
}

func (s *service) GetUserLastThreadTime(userID uint64) (*time.Time, error) {
	return s.userSvc.GetUserLastThreadTime(userID)
}

func (s *service) CreateThread(
	ctx context.Context,
	boardID uint64,
	sessionKey, title, content string,
	attachmentIDs []string,
) (*Thread, error) {
	titleLength := utf8.RuneCountInString(title)
	if titleLength < 3 || titleLength > 99 {
		return nil, fmt.Errorf("thread title must be between 3 and 99 characters, got %d", titleLength)
	}
	contentLength := utf8.RuneCountInString(content)
	if contentLength < 3 || contentLength > 999 {
		return nil, fmt.Errorf("thread content must be between 3 and 999 characters, got %d", contentLength)
	}
	user, err := s.sessionSvc.GetUserBySessionKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	lastThreadTime, err := s.GetUserLastThreadTime(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last thread time: %w", err)
	}
	if lastThreadTime != nil {
		elapsed := time.Since(*lastThreadTime)
		if elapsed < 5*time.Minute {
			secondsLeft := int64(300 - elapsed.Seconds())
			return nil, fmt.Errorf("thread creation cooldown: %d seconds left", secondsLeft)
		}
	}
	session, err := s.sessionSvc.GetSessionByKey(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	now := time.Now()
	var threadID uint64
	err = s.dbConn.Transaction(func(tx *gorm.DB) error {
		threadData := map[string]interface{}{
			"board_id":              boardID,
			"title":                 title,
			"content":               content,
			"created_by_session_id": session.ID,
			"author_nickname":       user.Nickname,
			"created_at":            now,
			"updated_at":            now,
		}

		if err := tx.Table("threads").Create(threadData).Error; err != nil {
			return err
		}

		if err := tx.Raw(`
            SELECT id FROM threads 
            WHERE created_by_session_id = ? AND created_at = ?
        `, session.ID, now).Scan(&threadID).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
            INSERT INTO user_activity (user_id, thread_count, last_thread_at)
            VALUES (?, 1, ?)
            ON CONFLICT (user_id) DO UPDATE SET
                thread_count = user_activity.thread_count + 1,
                last_thread_at = EXCLUDED.last_thread_at,
                updated_at = NOW()
        `, user.ID, now).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
            INSERT INTO threads_activity (thread_id, message_count, bump_at)
            VALUES (?, 0, NOW())
            ON CONFLICT (thread_id) DO NOTHING
        `, threadID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	if len(attachmentIDs) > 0 && s.attachmentSvc != nil {
		if err := s.attachmentSvc.LinkToThreadByFileID(ctx, attachmentIDs, threadID); err != nil {
			s.logger.Warn("Failed to link attachments to thread", zap.Uint64("thread_id", threadID), zap.Error(err))
		}
	}

	threadData, err := s.repo.GetThreadByID(threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created thread: %w", err)
	}

	s.invalidateCache(boardID)
	s.InvalidateTopThreadsCache()

	userCacheKey := fmt.Sprintf("user:session:%s", sessionKey)
	s.redisP.Del(context.Background(), userCacheKey)

	eventData := map[string]interface{}{
		"thread_id":       threadData.ID,
		"board_id":        threadData.BoardID,
		"title":           threadData.Title,
		"content":         threadData.Content,
		"created_at":      threadData.CreatedAt,
		"updated_at":      threadData.UpdatedAt,
		"created_by":      user.ID,
		"author_nickname": threadData.AuthorNickname,
		"messages_count":  threadData.MessagesCount,
		"timestamp":       time.Now().UTC().Unix(),
	}
	s.eventBus.Publish("thread_created", eventData)
	return threadData, nil
}

func (s *service) GetThreadsByBoardID(
	ctx context.Context,
	boardID uint64,
	sort string,
	page, limit int,
) ([]*Thread, int64, error) {
	validSorts := map[string]bool{"new": true, "popular": true, "active": true}
	if !validSorts[sort] {
		sort = "new"
	}

	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	cacheKey := fmt.Sprintf("%s:%d:sort:%s:page:%d:limit:%d", s.cachePrefix, boardID, sort, page, limit)

	cmd := s.redisP.Get(ctx, cacheKey)
	cachedData, err := cmd.Result()
	var result struct {
		Threads []*Thread `json:"threads"`
		Total   int64     `json:"total"`
	}
	if err == nil && cachedData != "" {
		if json.Unmarshal([]byte(cachedData), &result) == nil {
			return result.Threads, result.Total, nil
		}
	}

	threads, total, err := s.repo.GetThreadsByBoardID(boardID, sort, true, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get threads: %w", err)
	}

	if len(threads) > 0 && s.attachmentSvc != nil {
		for _, thread := range threads {
			attachments, err := s.attachmentSvc.GetByThreadID(ctx, thread.ID)
			if err != nil {
				s.logger.Warn("Failed to get attachments for thread",
					zap.Uint64("thread_id", thread.ID),
					zap.Error(err),
				)
				continue
			}
			if len(attachments) > 0 {
				thread.Attachments = make([]*ThreadAttachment, 0, len(attachments))
				for _, att := range attachments {
					thread.Attachments = append(thread.Attachments, &ThreadAttachment{
						ID:          fmt.Sprintf("%d", att.ID),
						FileID:      att.FileID,
						FileName:    att.FileName,
						FileURL:     att.FileURL,
						FileSize:    att.FileSize,
						ContentType: att.ContentType,
						ObjectName:  att.ObjectName,
						CreatedAt:   att.CreatedAt.Format(time.RFC3339),
					})
				}
			}
		}
	}

	if len(threads) > 0 {
		result.Threads = threads
		result.Total = total
		data, err := json.Marshal(result)
		if err == nil {
			s.redisP.SetEX(ctx, cacheKey, data, 5*time.Minute)
		}
	}
	return threads, total, nil
}

func (s *service) GetThreadByID(ctx context.Context, threadID uint64) (*Thread, error) {
	cacheKey := fmt.Sprintf("%s:thread:%d", s.cachePrefix, threadID)
	cmd := s.redisP.Get(ctx, cacheKey)
	cachedData, err := cmd.Result()
	var thread Thread
	if err == nil && cachedData != "" {
		if json.Unmarshal([]byte(cachedData), &thread) == nil {
			return &thread, nil
		}
	}

	threadData, err := s.repo.GetThreadByID(threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	if threadData != nil {
		if s.attachmentSvc != nil {
			attachments, err := s.attachmentSvc.GetByThreadID(ctx, threadID)
			if err != nil {
				s.logger.Warn("Failed to get attachments for thread",
					zap.Uint64("thread_id", threadID),
					zap.Error(err),
				)
			} else if len(attachments) > 0 {
				threadData.Attachments = make([]*ThreadAttachment, 0, len(attachments))
				for _, att := range attachments {
					threadData.Attachments = append(threadData.Attachments, &ThreadAttachment{
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
		data, err := json.Marshal(threadData)
		if err == nil {
			s.redisP.SetEX(ctx, cacheKey, data, 5*time.Minute)
		}
	}
	return threadData, nil
}

func (s *service) InvalidateThreadsCache(boardID uint64) {
	s.invalidateCache(boardID)
}

func (s *service) invalidateCache(boardID uint64) {
	ctx := context.Background()
	pattern := fmt.Sprintf("%s:%d:sort:*", s.cachePrefix, boardID)
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
		s.logger.Debugw("Thread list cache invalidated", "board_id", boardID, "deleted_keys", deletedCount)
	}
}

func (s *service) GetTopThreads(ctx context.Context, sort string, page, limit int) ([]*Thread, int64, error) {
	validSorts := map[string]bool{"new": true, "popular": true, "active": true}
	if !validSorts[sort] {
		sort = "new"
	}

	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	cacheKey := fmt.Sprintf("threads:top:sort:%s:page:%d:limit:%d", sort, page, limit)
	cmd := s.redisP.Get(ctx, cacheKey)
	cachedData, err := cmd.Result()
	var result struct {
		Threads []*Thread `json:"threads"`
		Total   int64     `json:"total"`
	}
	if err == nil && cachedData != "" {
		if json.Unmarshal([]byte(cachedData), &result) == nil {
			return result.Threads, result.Total, nil
		}
	}

	threads, total, err := s.repo.GetTopThreads(sort, page, limit)
	if err != nil {
		return nil, 0, err
	}

	for _, t := range threads {
		attachments, err := s.attachmentSvc.GetByThreadID(ctx, t.ID)
		if err != nil {
			s.logger.Warnw("Failed to load attachments for thread", "thread_id", t.ID, "error", err)
			continue
		}
		t.Attachments = make([]*ThreadAttachment, len(attachments))
		for i, att := range attachments {
			t.Attachments[i] = &ThreadAttachment{
				ID:          fmt.Sprintf("%d", att.ID),
				FileID:      att.FileID,
				FileName:    att.FileName,
				FileURL:     att.FileURL,
				FileSize:    att.FileSize,
				ContentType: att.ContentType,
				ObjectName:  att.ObjectName,
				CreatedAt:   att.CreatedAt.Format(time.RFC3339),
			}
		}
	}

	if len(threads) > 0 {
		result.Threads = threads
		result.Total = total
		data, _ := json.Marshal(result)
		s.redisP.SetEX(ctx, cacheKey, data, 5*time.Minute)
	}

	return threads, total, nil
}

func (s *service) InvalidateTopThreadsCache() {
	ctx := context.Background()
	pattern := "threads:top:sort:*:page:*:limit:*"
	var cursor uint64
	deletedCount := 0
	for {
		keys, cur, err := s.redisP.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			s.logger.Warnw("Redis scan failed during top threads cache invalidation", "error", err, "pattern", pattern)
			return
		}
		if len(keys) > 0 {
			n, err := s.redisP.Del(ctx, keys...).Result()
			if err != nil {
				s.logger.Warnw("Failed to delete top threads cache keys", "error", err, "keys", keys)
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
		s.logger.Debugw("Top threads cache invalidated", "deleted_keys", deletedCount)
	}
}

func (s *service) IsUserAuthor(ctx context.Context, userID uint64, threadID uint64) (bool, error) {
	return s.repo.IsUserThreadAuthor(userID, threadID)
}
