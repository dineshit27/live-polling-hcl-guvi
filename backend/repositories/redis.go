package repositories

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the go-redis client instance.
type RedisClient struct {
	Client *redis.Client
}

// ConnectRedis establishes a connection to Redis with retries and exponential backoff.
func ConnectRedis(ctx context.Context, addr, password string, db int) (*RedisClient, error) {
	opts := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	client := redis.NewClient(opts)

	maxRetries := 5
	backoff := 500 * time.Millisecond
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[Redis] Attempting connection to %s (attempt %d/%d)...", addr, attempt, maxRetries)

		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = client.Ping(pingCtx).Err()
		cancel()

		if err == nil {
			log.Printf("[Redis] Connected successfully to Redis instance at %s", addr)
			return &RedisClient{Client: client}, nil
		}

		log.Printf("[Redis] Connection attempt %d failed: %v", attempt, err)

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				_ = client.Close()
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}

	_ = client.Close()
	return nil, fmt.Errorf("failed to connect to Redis after %d attempts: %w", maxRetries, err)
}

// Ping verifies Redis liveness.
func (r *RedisClient) Ping(ctx context.Context) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return r.Client.Ping(ctx).Err()
}

// Close gracefully disconnects the Redis client.
func (r *RedisClient) Close() error {
	if r != nil && r.Client != nil {
		return r.Client.Close()
	}
	return nil
}
