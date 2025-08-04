package utils

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Services  []Service `json:"services"`
}

type Service struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type HealthChecker struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func (h *HealthChecker) Check(ctx context.Context) HealthStatus {
	var services []Service
	overallStatus := "healthy"

	if h.DB != nil {
		service := Service{Name: "PostgreSQL"}
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		sqlDB, _ := h.DB.DB()
		if err := sqlDB.PingContext(ctx); err != nil {
			service.Status = "down"
			service.Message = err.Error()
			overallStatus = "degraded"
		} else {
			service.Status = "up"
		}
		services = append(services, service)
		cancel()
	}

	if h.Redis != nil {
		service := Service{Name: "Redis"}
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if err := h.Redis.Ping(ctx).Err(); err != nil {
			service.Status = "down"
			service.Message = err.Error()
			overallStatus = "degraded"
		} else {
			service.Status = "up"
		}
		services = append(services, service)
		cancel()
	}

	return HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now().UTC(),
		Services:  services,
	}
}
