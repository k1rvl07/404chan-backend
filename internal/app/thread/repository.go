package thread

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	GetThreadsByBoardID(boardID uint64, sort string, last24Hours bool, page int, limit int) ([]*Thread, int64, error)
	GetThreadByID(id uint64) (*Thread, error)
	GetUserLastThreadTime(userID uint64) (*time.Time, error)
	GetTotalThreadsCount(boardID uint64) (int64, error)
	GetTopThreads(sort string, page, limit int) ([]*Thread, int64, error)
	IsUserThreadAuthor(userID uint64, threadID uint64) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetThreadsByBoardID(boardID uint64, sort string, last24Hours bool, page int, limit int) ([]*Thread, int64, error) {
	var threads []*Thread

	query := r.db.Table("threads").
		Select(`
			threads.id, 
			threads.board_id, 
			boards.slug as board_slug, 
			threads.title, 
			threads.content, 
			threads.created_at, 
			threads.updated_at, 
			users.id as created_by, 
			threads.author_nickname as author_nickname, 
			COALESCE(threads_activity.message_count, 0) as messages_count, 
			threads_activity.bump_at
		`).
		Joins("JOIN sessions ON sessions.id = threads.created_by_session_id").
		Joins("JOIN users ON users.id = sessions.user_id").
		Joins("JOIN boards ON boards.id = threads.board_id").
		Joins("LEFT JOIN threads_activity ON threads_activity.thread_id = threads.id").
		Where("threads.board_id = ?", boardID)

	if last24Hours {
		query = query.Where("threads.created_at > NOW() - INTERVAL '24 hours'")
	}

	switch sort {
	case "popular":
		query = query.Order("threads_activity.message_count DESC")
	case "active":
		query = query.Order("threads_activity.bump_at DESC")
	default:
		query = query.Order("threads.created_at DESC")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit).Group("threads.id, boards.slug, users.id, threads_activity.message_count, threads_activity.bump_at")

	if err := query.Find(&threads).Error; err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}

func (r *repository) GetThreadByID(id uint64) (*Thread, error) {
	var thread Thread
	err := r.db.Table("threads").
		Select(`
			threads.*, 
			boards.slug as board_slug, 
			threads.author_nickname as author_nickname,
			users.id as created_by,
			COALESCE(threads_activity.message_count, 0) as messages_count
		`).
		Joins("JOIN sessions ON sessions.id = threads.created_by_session_id").
		Joins("JOIN users ON users.id = sessions.user_id").
		Joins("JOIN boards ON boards.id = threads.board_id").
		Joins("LEFT JOIN threads_activity ON threads_activity.thread_id = threads.id").
		Where("threads.id = ?", id).
		First(&thread).Error
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

func (r *repository) GetUserLastThreadTime(userID uint64) (*time.Time, error) {
	var nullTime sql.NullTime
	err := r.db.Model(&Thread{}).
		Select("MAX(threads.created_at)").
		Joins("JOIN sessions ON sessions.id = threads.created_by_session_id").
		Joins("JOIN users ON users.id = sessions.user_id").
		Where("users.id = ?", userID).
		Scan(&nullTime).Error
	if err != nil {
		return nil, err
	}
	if !nullTime.Valid {
		return nil, nil
	}
	return &nullTime.Time, nil
}

func (r *repository) GetTotalThreadsCount(boardID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&Thread{}).
		Where("board_id = ?", boardID).
		Count(&count).Error
	return count, err
}

func (r *repository) GetTopThreads(sort string, page, limit int) ([]*Thread, int64, error) {
	var threads []*Thread

	query := r.db.Table("threads").
		Select(`
			threads.id, 
			threads.board_id, 
			boards.slug as board_slug, 
			threads.title, 
			threads.content, 
			threads.created_at, 
			threads.updated_at, 
			users.id as created_by, 
			threads.author_nickname as author_nickname, 
			COALESCE(threads_activity.message_count, 0) as messages_count, 
			threads_activity.bump_at
		`).
		Joins("JOIN sessions ON sessions.id = threads.created_by_session_id").
		Joins("JOIN users ON users.id = sessions.user_id").
		Joins("JOIN boards ON boards.id = threads.board_id").
		Joins("LEFT JOIN threads_activity ON threads_activity.thread_id = threads.id")

	switch sort {
	case "popular":
		query = query.Order("threads_activity.message_count DESC")
	case "active":
		query = query.Order("threads_activity.bump_at DESC")
	default:
		query = query.Order("threads.created_at DESC")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit).Group("threads.id, boards.slug, users.id, threads_activity.message_count, threads_activity.bump_at")

	if err := query.Find(&threads).Error; err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}

func (r *repository) IsUserThreadAuthor(userID uint64, threadID uint64) (bool, error) {
	var count int64
	err := r.db.Table("threads").
		Joins("JOIN sessions ON sessions.id = threads.created_by_session_id").
		Joins("JOIN users ON users.id = sessions.user_id").
		Where("threads.id = ? AND users.id = ?", threadID, userID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
