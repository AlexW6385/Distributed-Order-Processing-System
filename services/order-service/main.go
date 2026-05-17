package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	paymentv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/payment/v1"
	productv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/product/v1"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/cache"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/config"
	appdb "github.com/AlexW6385/Distributed-Order-Processing-System/internal/db"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/events"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/health"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/observability"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/order"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/product"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	productConn, err := grpc.NewClient(
		cfg.ProductServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observability.UnaryClientInterceptor("order-service")),
	)
	if err != nil {
		log.Fatalf("connect product-service: %v", err)
	}
	defer productConn.Close()

	paymentConn, err := grpc.NewClient(
		cfg.PaymentServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observability.UnaryClientInterceptor("order-service")),
	)
	if err != nil {
		log.Fatalf("connect payment-service: %v", err)
	}
	defer paymentConn.Close()

	productGRPCClient := productv1.NewProductServiceClient(productConn)
	paymentGRPCClient := paymentv1.NewPaymentServiceClient(paymentConn)

	productHTTPClient := product.NewGRPCClient(productGRPCClient)
	orderProductClient := order.NewProductGRPCClient(productGRPCClient)
	orderPaymentClient := order.NewPaymentGRPCClient(paymentGRPCClient)

	orderRepository := order.NewRepository(db)
	orderOptions := []order.Option{
		order.WithProductClient(orderProductClient),
		order.WithPaymentClient(orderPaymentClient),
	}
	if len(cfg.KafkaBrokers) > 0 {
		if err := events.EnsureTopic(context.Background(), cfg.KafkaBrokers, cfg.OrderPaidTopic); err != nil {
			log.Fatalf("ensure kafka topic: %v", err)
		}
		orderPaidPublisher := events.NewKafkaOrderPaidPublisher(cfg.KafkaBrokers, cfg.OrderPaidTopic)
		defer orderPaidPublisher.Close()

		outboxRepository := events.NewOutboxRepository(db)
		outboxPublisher := events.NewOutboxPublisher(outboxRepository, orderPaidPublisher, 2*time.Second, 10)
		go outboxPublisher.Run(context.Background())
	}
	orderService := order.NewService(orderRepository, orderOptions...)

	router := gin.New()
	router.Use(gin.Recovery(), observability.HTTPMiddleware("order-service"))
	health.NewHandler(db, redisClient).RegisterRoutes(router)
	newProductHTTPHandler(productHTTPClient).registerRoutes(router)
	order.NewHandler(orderService).RegisterRoutes(router)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("order-service listening", slog.String("port", cfg.Port))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}
