package board

import "gorm.io/gorm"

type Repository interface {
	GetAllBoards() ([]*Board, error)
	GetBoardBySlug(slug string) (*Board, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetAllBoards() ([]*Board, error) {
	var boards []*Board
	err := r.db.
		Order("created_at ASC").
		Find(&boards).Error
	return boards, err
}

func (r *repository) GetBoardBySlug(slug string) (*Board, error) {
	var board Board
	err := r.db.Where("slug = ?", slug).First(&board).Error
	return &board, err
}
