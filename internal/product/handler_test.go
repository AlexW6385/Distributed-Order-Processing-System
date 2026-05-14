package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeProductService struct {
	products []Product
	err      error
}

func (s *fakeProductService) List(ctx context.Context) ([]Product, error) {
	return s.products, s.err
}

func TestHandlerListProductsReturnsProducts(t *testing.T) {
	router := productTestRouter(&fakeProductService{
		products: []Product{{ID: "product-1", SKU: "keyboard-001", Name: "Keyboard"}},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/products", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload struct {
		Products []Product `json:"products"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(payload.Products))
	}
}

func TestHandlerListProductsReturnsServerError(t *testing.T) {
	router := productTestRouter(&fakeProductService{err: errors.New("database unavailable")})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/products", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}
}

func productTestRouter(service ProductService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(service).RegisterRoutes(router)
	return router
}
