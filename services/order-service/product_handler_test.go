package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/product"
	"github.com/gin-gonic/gin"
)

type fakeProductListService struct {
	products []product.Product
	err      error
}

func (s *fakeProductListService) List(ctx context.Context) ([]product.Product, error) {
	return s.products, s.err
}

func TestProductHTTPHandlerListProductsReturnsProducts(t *testing.T) {
	router := productTestRouter(&fakeProductListService{
		products: []product.Product{{ID: "product-1", SKU: "keyboard-001", Name: "Keyboard"}},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/products", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload struct {
		Products []product.Product `json:"products"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(payload.Products))
	}
}

func TestProductHTTPHandlerListProductsReturnsServerError(t *testing.T) {
	router := productTestRouter(&fakeProductListService{err: errors.New("database unavailable")})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/products", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}
}

func productTestRouter(service productListService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	newProductHTTPHandler(service).registerRoutes(router)
	return router
}
