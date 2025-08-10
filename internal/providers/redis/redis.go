package redis

import (
	"context"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisProvider struct {
	Client *redis.Client
	URL    string
	logger *zap.SugaredLogger
}

func NewRedisProvider(redisURL string, logger *zap.Logger) *RedisProvider {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	provider := &RedisProvider{
		Client: client,
		URL:    redisURL,
		logger: logger.Sugar(),
	}

	client.AddHook(&loggerHook{provider: provider})

	go provider.startConnectionMonitor(context.Background())

	provider.logger.Infow("Redis connected", "url", redisURL)
	return provider
}

func (r *RedisProvider) startConnectionMonitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var wasConnected bool

	if err := r.Client.Ping(ctx).Err(); err == nil {
		wasConnected = true
	} else {
		r.logger.Warnw("Redis unavailable at startup", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := r.Client.Ping(ctx).Err()
			if err != nil {
				if wasConnected {
					r.logger.Errorw("Redis disconnected", "error", err)
					wasConnected = false
				}
			} else {
				if !wasConnected {
					r.logger.Infow("Redis reconnected", "url", r.URL)
					wasConnected = true
				}
			}
		}
	}
}

type loggerHook struct {
	provider *RedisProvider
}

func (h *loggerHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *loggerHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if err != nil {
			h.provider.logger.Warnw("Redis command failed",
				"command", cmd.Name(),
				"error", err,
			)
		}
		return err
	}
}

func (h *loggerHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		err := next(ctx, cmds)
		if err != nil {
			h.provider.logger.Warnw("Redis pipeline failed", "error", err)
		}
		return err
	}
}
