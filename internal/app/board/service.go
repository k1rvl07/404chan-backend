package board

type Service interface {
	GetAllBoards() ([]*Board, error)
	GetBoardBySlug(slug string) (*Board, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetAllBoards() ([]*Board, error) {
	return s.repo.GetAllBoards()
}

func (s *service) GetBoardBySlug(slug string) (*Board, error) {
	return s.repo.GetBoardBySlug(slug)
}
