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
	ttl    time.Duration
}

func NewRedisProvider(redisURL string, logger *zap.Logger, ttl time.Duration) *RedisProvider {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		opts = &redis.Options{
			Addr: redisURL,
			DB:   0,
		}
	}

	client := redis.NewClient(opts)

	client.Options().MaxRetries = 3
	client.Options().MinRetryBackoff = 100 * time.Millisecond
	client.Options().MaxRetryBackoff = 500 * time.Millisecond

	provider := &RedisProvider{
		Client: client,
		URL:    redisURL,
		logger: logger.Sugar(),
		ttl:    ttl,
	}

	client.AddHook(&loggerHook{provider: provider})

	go provider.startConnectionMonitor(context.Background())

	if err := client.Ping(context.Background()).Err(); err != nil {
		provider.logger.Errorw("Redis connection failed at startup", "error", err)
	} else {
		provider.logger.Infow("Redis connected",
			"url", redisURL,
			"db", opts.DB,
			"username", opts.Username,
			"default_ttl", ttl.String(),
		)
	}

	return provider
}

func (r *RedisProvider) SetWithDefaultTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	if ttl <= 0 {
		ttl = r.ttl
	}
	return r.Client.Set(ctx, key, value, ttl)
}

func (r *RedisProvider) SetEX(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	return r.Client.Set(ctx, key, value, ttl)
}

func (r *RedisProvider) Get(ctx context.Context, key string) *redis.StringCmd {
	return r.Client.Get(ctx, key)
}

func (r *RedisProvider) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return r.Client.Del(ctx, keys...)
}

func (r *RedisProvider) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return r.Client.Exists(ctx, keys...)
}

func (r *RedisProvider) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return r.Client.Keys(ctx, pattern)
}

func (r *RedisProvider) Scan(ctx context.Context, cursor uint64, pattern string, count int64) *redis.ScanCmd {
	return r.Client.Scan(ctx, cursor, pattern, count)
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
		conn, err := next(ctx, network, addr)
		if err != nil {
			h.provider.logger.Errorw("Redis dial failed", "network", network, "addr", addr, "error", err)
		} else {
			h.provider.logger.Debugw("Redis dialed", "network", network, "addr", addr)
		}
		return conn, err
	}
}

func (h *loggerHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		duration := time.Since(start)

		if cmd.Name() == "ping" && err == nil {
			return err
		}

		fields := []interface{}{
			"command", cmd.Name(),
			"args", cmd.Args(),
			"duration_ms", duration.Milliseconds(),
			"duration", duration.String(),
		}
		if err != nil {
			fields = append(fields, "error", err)
		}

		if err != nil {
			h.provider.logger.Errorw("Redis command failed", fields...)
		} else {
			h.provider.logger.Debugw("Redis command executed", fields...)
		}

		return err
	}
}

func (h *loggerHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		duration := time.Since(start)

		for _, cmd := range cmds {
			if cmd.Name() == "ping" && err == nil {
				continue
			}

			fields := []interface{}{
				"command", cmd.Name(),
				"args", cmd.Args(),
				"duration_ms", duration.Milliseconds(),
				"duration", duration.String(),
			}
			if err != nil {
				fields = append(fields, "error", err)
			}

			if err != nil {
				h.provider.logger.Errorw("Redis pipeline command failed", fields...)
			} else {
				h.provider.logger.Debugw("Redis pipeline command executed", fields...)
			}
		}

		return err
	}
}
