package product

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const productsCacheKey = "products:all"

type Cache interface {
	GetProducts(ctx context.Context) ([]Product, bool, error)
	SetProducts(ctx context.Context, products []Product) error
	DeleteProducts(ctx context.Context) error
}

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
		ttl:    5 * time.Minute,
	}
}

func (c *RedisCache) GetProducts(ctx context.Context) ([]Product, bool, error) {
	data, err := c.client.Get(ctx, productsCacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var products []Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, false, err
	}

	return products, true, nil
}

func (c *RedisCache) SetProducts(ctx context.Context, products []Product) error {
	data, err := json.Marshal(products)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, productsCacheKey, data, c.ttl).Err()
}

func (c *RedisCache) DeleteProducts(ctx context.Context) error {
	return c.client.Del(ctx, productsCacheKey).Err()
}
