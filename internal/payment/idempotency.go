package payment

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const paymentIdempotencyPrefix = "idempotency:payment:"

type IdempotencyStore interface {
	ReservePayment(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleasePayment(ctx context.Context, key string) error
}

type RedisIdempotencyStore struct {
	client *redis.Client
}

func NewRedisIdempotencyStore(client *redis.Client) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{client: client}
}

func (s *RedisIdempotencyStore) ReservePayment(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, paymentIdempotencyPrefix+key, "reserved", ttl).Result()
}

func (s *RedisIdempotencyStore) ReleasePayment(ctx context.Context, key string) error {
	return s.client.Del(ctx, paymentIdempotencyPrefix+key).Err()
}
