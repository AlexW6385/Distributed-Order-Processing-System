package main

import (
	"context"
	"net/http"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/product"
	"github.com/gin-gonic/gin"
)

type productHTTPHandler struct {
	service productListService
}

type productListService interface {
	List(ctx context.Context) ([]product.Product, error)
}

func newProductHTTPHandler(service productListService) *productHTTPHandler {
	return &productHTTPHandler{service: service}
}

func (h *productHTTPHandler) registerRoutes(router gin.IRoutes) {
	router.GET("/products", h.list)
}

func (h *productHTTPHandler) list(c *gin.Context) {
	products, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}
