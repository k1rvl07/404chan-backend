package thread

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"backend/internal/app/session"
	"backend/internal/app/user"
	"backend/internal/providers/redis"
	"backend/internal/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service interface {
	CreateThread(ctx context.Context, boardID uint64, sessionKey string, title string, content string) (*Thread, error)
	GetThreadsByBoardID(ctx context.Context, boardID uint64, sort string, page int, limit int) ([]*Thread, int64, error)
	GetUserLastThreadTime(userID uint64) (*time.Time, error)
}

type service struct {
	repo        Repository
	sessionSvc  session.Service
	userSvc     user.Service
	dbConn      *gorm.DB
	redisP      *redis.RedisProvider
	eventBus    *utils.EventBus
	logger      *zap.SugaredLogger
	cachePrefix string
}

func NewService(repo Repository, sessionSvc session.Service, userSvc user.Service, dbConn *gorm.DB, redisP *redis.RedisProvider, eventBus *utils.EventBus, logger *zap.Logger) Service {
	return &service{
		repo:        repo,
		sessionSvc:  sessionSvc,
		userSvc:     userSvc,
		dbConn:      dbConn,
		redisP:      redisP,
		eventBus:    eventBus,
		logger:      logger.Sugar(),
		cachePrefix: "threads:board",
	}
}

func (s *service) GetUserLastThreadTime(userID uint64) (*time.Time, error) {
	return s.userSvc.GetUserLastThreadTime(userID)
}

func (s *service) CreateThread(ctx context.Context, boardID uint64, sessionKey string, title string, content string) (*Thread, error) {
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
			return nil, fmt.Errorf("thread creation cooldown: %d seconds left", int64(300-elapsed.Seconds()))
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
            INSERT INTO user_activities (user_id, thread_count, last_thread_at)
            VALUES (?, 1, ?)
            ON CONFLICT (user_id) DO UPDATE SET
                thread_count = user_activities.thread_count + 1,
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

	threadData, err := s.repo.GetThreadByID(threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created thread: %w", err)
	}

	s.invalidateCache(boardID)

	eventData := map[string]interface{}{
		"id":             threadData.ID,
		"board_id":       threadData.BoardID,
		"title":          threadData.Title,
		"content":        threadData.Content,
		"created_at":     threadData.CreatedAt,
		"updated_at":     threadData.UpdatedAt,
		"created_by":     threadData.CreatedBy,
		"authorNickname": threadData.AuthorNickname,
		"messages_count": threadData.MessagesCount,
		"timestamp":      time.Now().UTC().Unix(),
	}

	s.eventBus.Publish("thread_created", eventData)

	return threadData, nil
}

func (s *service) GetThreadsByBoardID(ctx context.Context, boardID uint64, sort string, page int, limit int) ([]*Thread, int64, error) {
	validSorts := map[string]bool{"new": true, "popular": true, "active": true}
	if !validSorts[sort] {
		sort = "new"
	}

	cacheKey := fmt.Sprintf("%s:%d:sort:%s:page:%d:limit:%d", s.cachePrefix, boardID, sort, page, limit)

	cmd := s.redisP.Get(ctx, cacheKey)
	cachedData, err := cmd.Result()
	var result struct {
		Threads []*Thread `json:"threads"`
		Total   int64     `json:"total"`
	}

	if err == nil && cachedData != "" {
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return result.Threads, result.Total, nil
		}
	}

	threads, total, err := s.repo.GetThreadsByBoardID(boardID, sort, true, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get threads: %w", err)
	}

	if len(threads) > 0 {
		result.Threads = threads
		result.Total = total
		data, err := json.Marshal(result)
		if err == nil {
			s.redisP.SetWithDefaultTTL(ctx, cacheKey, data, 0)
		}
	}

	return threads, total, nil
}

func (s *service) invalidateCache(boardID uint64) {
	sorts := []string{"new", "popular", "active"}
	for _, sort := range sorts {
		for page := 1; page <= 10; page++ {
			for _, limit := range []int{10, 20, 50} {
				cacheKey := fmt.Sprintf("%s:%d:sort:%s:page:%d:limit:%d", s.cachePrefix, boardID, sort, page, limit)
				s.redisP.Del(context.Background(), cacheKey)
			}
		}
	}
}
