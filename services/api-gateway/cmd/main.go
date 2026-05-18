package main

import (
	"fmt"
	"strconv"
	"time"

	orderv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/order/v1"
	productv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/product/v1"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/platform/config"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/platform/logging"
	"github.com/AlexW6385/Distributed-Order-Processing-System/services/api-gateway/internal/httpapi"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log := logging.New("api-gateway")

	orderConn, err := grpc.NewClient(config.String("ORDER_GRPC_ADDR", "localhost:50051"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("order grpc connection failed", err, nil)
		panic(err)
	}
	defer orderConn.Close()

	productConn, err := grpc.NewClient(config.String("PRODUCT_GRPC_ADDR", "localhost:50052"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("product grpc connection failed", err, nil)
		panic(err)
	}
	defer productConn.Close()

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpapi.RateLimit(httpapi.RateLimitConfig{
		RequestsPerSecond: configFloat("RATE_LIMIT_RPS", 2),
		Burst:             configInt("RATE_LIMIT_BURST", 10),
		IdleTTL:           10 * time.Minute,
	}))
	httpapi.New(orderv1.NewOrderServiceClient(orderConn), productv1.NewProductServiceClient(productConn)).Register(router)

	addr := fmt.Sprintf(":%s", config.String("API_GATEWAY_PORT", "8080"))
	log.Info("http server started", map[string]any{"addr": addr})
	if err := router.Run(addr); err != nil {
		log.Error("http server stopped", err, nil)
	}
}

func configFloat(key string, fallback float64) float64 {
	raw := config.String(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func configInt(key string, fallback int) int {
	raw := config.String(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
