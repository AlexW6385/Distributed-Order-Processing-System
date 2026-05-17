package main

import (
	"log"
	"log/slog"
	"os"

	paymentv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/payment/v1"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/cache"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/config"
	appdb "github.com/AlexW6385/Distributed-Order-Processing-System/internal/db"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/observability"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/payment"
	"google.golang.org/grpc"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
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

	paymentRepository := payment.NewRepository(db)
	idempotencyStore := payment.NewRedisIdempotencyStore(redisClient)
	paymentService := payment.NewService(paymentRepository, idempotencyStore)

	listener, err := listenGRPC(cfg.GRPCPort)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(observability.UnaryServerInterceptor("payment-service")))
	paymentv1.RegisterPaymentServiceServer(server, newPaymentServer(paymentService))

	slog.Info("payment-service listening", slog.String("port", cfg.GRPCPort))
	if err := server.Serve(listener); err != nil {
		log.Fatalf("serve grpc: %v", err)
	}
}
