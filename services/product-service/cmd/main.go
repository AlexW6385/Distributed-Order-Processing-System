package main

import (
	"context"
	"fmt"
	"net"
	"time"

	productv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/product/v1"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/platform/config"
	platformdb "github.com/AlexW6385/Distributed-Order-Processing-System/internal/platform/db"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/platform/logging"
	productcache "github.com/AlexW6385/Distributed-Order-Processing-System/services/product-service/internal/cache"
	"github.com/AlexW6385/Distributed-Order-Processing-System/services/product-service/internal/repository"
	"github.com/AlexW6385/Distributed-Order-Processing-System/services/product-service/internal/service"
	"github.com/AlexW6385/Distributed-Order-Processing-System/services/product-service/internal/transport/grpcapi"
	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	log := logging.New("product-service")

	db, err := platformdb.Open(ctx, config.String("DATABASE_URL", "postgres://orders:orders@localhost:5432/orders?sslmode=disable"))
	if err != nil {
		log.Error("database connection failed", err, nil)
		panic(err)
	}
	defer db.Close()

	var cache service.ProductCache
	redisClient := redis.NewClient(&redis.Options{Addr: config.String("REDIS_ADDR", "localhost:6379")})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error("redis cache disabled", err, nil)
		_ = redisClient.Close()
	} else {
		defer redisClient.Close()
		cache = productcache.NewRedis(redisClient, 30*time.Second)
	}

	port := config.String("PRODUCT_GRPC_PORT", "50052")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Error("listen failed", err, nil)
		panic(err)
	}

	server := grpc.NewServer()
	productv1.RegisterProductServiceServer(server, grpcapi.New(service.New(repository.New(db), cache)))
	log.Info("grpc server started", map[string]any{"addr": fmt.Sprintf(":%s", port)})
	if err := server.Serve(listener); err != nil {
		log.Error("grpc server stopped", err, nil)
	}
}
