package health

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	db          *sql.DB
	redisClient *redis.Client
}

func NewHandler(db *sql.DB, redisClient *redis.Client) *Handler {
	return &Handler{
		db:          db,
		redisClient: redisClient,
	}
}

func (h *Handler) RegisterRoutes(router gin.IRoutes) {
	router.GET("/health", h.check)
}

func (h *Handler) check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := http.StatusOK
	response := gin.H{
		"status":   "ok",
		"database": "ok",
		"redis":    "ok",
	}

	if err := h.db.PingContext(ctx); err != nil {
		status = http.StatusServiceUnavailable
		response["status"] = "unhealthy"
		response["database"] = "unavailable"
	}

	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		status = http.StatusServiceUnavailable
		response["status"] = "unhealthy"
		response["redis"] = "unavailable"
	}

	c.JSON(status, response)
}
