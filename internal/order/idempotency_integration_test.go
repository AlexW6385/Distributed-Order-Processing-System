package order

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/cache"
)

func TestRedisIdempotencyStoreReservesAndReleasesPaymentKeys(t *testing.T) {
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}

	client, err := cache.OpenRedis(redisAddr)
	if err != nil {
		t.Fatalf("open redis: %v", err)
	}

	store := NewRedisIdempotencyStore(client)
	key := "test-payment-" + time.Now().Format("20060102150405.000000000")
	t.Cleanup(func() {
		_ = store.ReleasePayment(context.Background(), key)
		client.Close()
	})

	reserved, err := store.ReservePayment(context.Background(), key, time.Minute)
	if err != nil {
		t.Fatalf("reserve payment: %v", err)
	}
	if !reserved {
		t.Fatal("expected first reservation to succeed")
	}

	reserved, err = store.ReservePayment(context.Background(), key, time.Minute)
	if err != nil {
		t.Fatalf("reserve duplicate payment: %v", err)
	}
	if reserved {
		t.Fatal("expected duplicate reservation to fail")
	}

	if err := store.ReleasePayment(context.Background(), key); err != nil {
		t.Fatalf("release payment: %v", err)
	}

	reserved, err = store.ReservePayment(context.Background(), key, time.Minute)
	if err != nil {
		t.Fatalf("reserve after release: %v", err)
	}
	if !reserved {
		t.Fatal("expected reservation after release to succeed")
	}
}
