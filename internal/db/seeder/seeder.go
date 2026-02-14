package seeder

import (
	"backend/internal/app/board"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Seeder struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewSeeder(db *gorm.DB, logger *zap.Logger) *Seeder {
	return &Seeder{
		db:     db,
		logger: logger,
	}
}

func (s *Seeder) Seed() error {
	s.logger.Info("Running database seeders...")

	if err := s.seedBoards(); err != nil {
		return err
	}

	s.logger.Info("Database seeders completed successfully")
	return nil
}

func (s *Seeder) seedBoards() error {
	var count int64
	s.db.Model(&board.Board{}).Count(&count)
	if count > 0 {
		s.logger.Info("Boards already exist, skipping seed")
		return nil
	}

	boards := []board.Board{
		{Slug: "a", Title: "Anime & Manga", Description: ptr("Аниме и манга")},
		{Slug: "b", Title: "Random", Description: ptr("Random")},
		{Slug: "c", Title: "Cute", Description: ptr("Милота")},
		{Slug: "mu", Title: "Music", Description: ptr("Музыка")},
		{Slug: "prog", Title: "Programming", Description: ptr("Программирование")},
		{Slug: "sci", Title: "Science", Description: ptr("Наука")},
	}

	if err := s.db.Create(&boards).Error; err != nil {
		return err
	}

	s.logger.Info("Seeded boards", zap.Int("count", len(boards)))
	return nil
}

func ptr(s string) *string {
	return &s
}
