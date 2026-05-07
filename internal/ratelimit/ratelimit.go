package ratelimit

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client         *redis.Client
	maxRunningJobs int
}

func NewRateLimiter(redisURL string, maxRunningJobs int) (*RateLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(opts)
	return &RateLimiter{client: client, maxRunningJobs: maxRunningJobs}, nil
}

func (r *RateLimiter) AllowJobSubmission(ctx context.Context, workspaceID string) (bool, error) {
	key := fmt.Sprintf("rate_limit:jobs:running:%s", workspaceID)

	count, err := r.client.Get(ctx, key).Int()
	if err == redis.Nil {
		count = 0
	} else if err != nil {
		return false, fmt.Errorf("failed to get rate limit counter: %w", err)
	}

	if count >= r.maxRunningJobs {
		return false, nil
	}

	return true, nil
}

func (r *RateLimiter) IncrementRunningJobs(ctx context.Context, workspaceID string) error {
	key := fmt.Sprintf("rate_limit:jobs:running:%s", workspaceID)
	return r.client.Incr(ctx, key).Err()
}

func (r *RateLimiter) DecrementRunningJobs(ctx context.Context, workspaceID string) error {
	key := fmt.Sprintf("rate_limit:jobs:running:%s", workspaceID)
	return r.client.Decr(ctx, key).Err()
}
