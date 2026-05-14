package product

import (
	"context"
	"os"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/cache"
)

func TestRedisCacheStoresAndLoadsProducts(t *testing.T) {
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}

	client, err := cache.OpenRedis(redisAddr)
	if err != nil {
		t.Fatalf("open redis: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Del(context.Background(), productsCacheKey).Err()
		client.Close()
	})

	productCache := NewRedisCache(client)
	products := []Product{{ID: "product-1", SKU: "keyboard-001", Name: "Keyboard"}}

	if err := productCache.SetProducts(context.Background(), products); err != nil {
		t.Fatalf("set products: %v", err)
	}

	cachedProducts, ok, err := productCache.GetProducts(context.Background())
	if err != nil {
		t.Fatalf("get products: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(cachedProducts) != 1 || cachedProducts[0].ID != "product-1" {
		t.Fatalf("expected cached product, got %+v", cachedProducts)
	}
}
