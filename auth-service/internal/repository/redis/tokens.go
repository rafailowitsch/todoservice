package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
	"todoservice/auth-service/internal/domain"
)

type TokenCache struct {
	cache *redis.Client
}

func NewTokenCache(cache *redis.Client) *TokenCache {
	return &TokenCache{
		cache: cache,
	}
}

func (t *TokenCache) Set(ctx context.Context, token *domain.RefreshToken) error {
	return t.cache.Set(ctx, token.ID.String(), token.RefreshToken, token.ExpiresAt.Sub(time.Now())).Err()
}
