package main

import (
	"log"
	"net/http"
	"time"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/config"
	appdb "github.com/AlexW6385/Distributed-Order-Processing-System/internal/db"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/health"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/order"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/product"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := appdb.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	productRepository := product.NewRepository(db)
	productService := product.NewService(productRepository)

	orderRepository := order.NewRepository(db)
	orderService := order.NewService(orderRepository)

	router := gin.Default()
	health.NewHandler(db).RegisterRoutes(router)
	product.NewHandler(productService).RegisterRoutes(router)
	order.NewHandler(orderService).RegisterRoutes(router)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("server listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}
