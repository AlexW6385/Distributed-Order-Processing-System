package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type app struct {
	db *sql.DB
}

func main() {
	databaseURL := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/distributed_order_processing_system?sslmode=disable")
	port := getenv("PORT", "8080")

	db, err := openDB(databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	application := &app{db: db}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           application.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("server listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}

func openDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func (a *app) routes() http.Handler {
	router := gin.Default()
	router.GET("/health", a.healthCheck)
	router.GET("/products", a.listProducts)
	router.POST("/orders", a.createOrder)
	router.GET("/orders/:id", a.getOrder)
	router.POST("/orders/:id/pay", a.payOrder)
	return router
}

func (a *app) healthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := http.StatusOK
	response := gin.H{
		"status":   "ok",
		"database": "ok",
	}

	if err := a.db.PingContext(ctx); err != nil {
		status = http.StatusServiceUnavailable
		response["status"] = "unhealthy"
		response["database"] = "unavailable"
	}

	c.JSON(status, response)
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
