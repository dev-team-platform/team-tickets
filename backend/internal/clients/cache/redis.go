package cache

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/trannhanlv2004/team-tickets/internal/configs"
)

// ConnectRedis creates and verifies a Redis client. The caller must close it.
func ConnectRedis(ctx context.Context, cfg configs.RedisConfig) (*redis.Client, error) {
	return connectRedis(ctx, &redis.Options{
		Addr:     net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Password: cfg.Password,
	})
}

func connectRedis(ctx context.Context, options *redis.Options) (*redis.Client, error) {
	client := redis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}
