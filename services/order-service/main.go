package main

import (
	"log"
	"net/http"
	"time"

	paymentv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/payment/v1"
	productv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/product/v1"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/cache"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/config"
	appdb "github.com/AlexW6385/Distributed-Order-Processing-System/internal/db"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/health"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/order"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/product"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	productConn, err := grpc.NewClient(cfg.ProductServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect product-service: %v", err)
	}
	defer productConn.Close()

	paymentConn, err := grpc.NewClient(cfg.PaymentServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	orderService := order.NewService(
		orderRepository,
		order.WithProductClient(orderProductClient),
		order.WithPaymentClient(orderPaymentClient),
	)

	router := gin.Default()
	health.NewHandler(db, redisClient).RegisterRoutes(router)
	product.NewHandler(productHTTPClient).RegisterRoutes(router)
	order.NewHandler(orderService).RegisterRoutes(router)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("order-service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}
