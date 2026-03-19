package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewRedis creates a Redis client and verifies the connection.
func NewRedis(ctx context.Context, redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("store: ping redis: %w", err)
	}

	return client, nil
}
