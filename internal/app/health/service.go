package health

import (
	"context"

	"backend/internal/utils"
)

type HealthService struct {
	checker *utils.HealthChecker
}

func NewHealthService(checker *utils.HealthChecker) *HealthService {
	return &HealthService{checker: checker}
}

func (s *HealthService) Check(ctx context.Context) utils.HealthStatus {
	return s.checker.Check(ctx)
}
