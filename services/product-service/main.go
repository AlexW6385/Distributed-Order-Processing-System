package main

import (
	"log"

	productv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/product/v1"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/cache"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/config"
	appdb "github.com/AlexW6385/Distributed-Order-Processing-System/internal/db"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/product"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	db, err := appdb.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	redisClient, err := cache.OpenRedis(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	productRepository := product.NewRepository(db)
	productCache := product.NewRedisCache(redisClient)
	productService := product.NewService(productRepository, productCache)

	listener, err := listenGRPC(cfg.GRPCPort)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}

	server := grpc.NewServer()
	productv1.RegisterProductServiceServer(server, newProductServer(productService))

	log.Printf("product-service listening on :%s", cfg.GRPCPort)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("serve grpc: %v", err)
	}
}
