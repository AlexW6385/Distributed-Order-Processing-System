package health

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
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
	}

	if err := h.db.PingContext(ctx); err != nil {
		status = http.StatusServiceUnavailable
		response["status"] = "unhealthy"
		response["database"] = "unavailable"
	}

	c.JSON(status, response)
}
