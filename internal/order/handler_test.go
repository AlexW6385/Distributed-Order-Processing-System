package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeOrderService struct {
	createOrder Order
	createErr   error
	getOrder    Order
	getErr      error
	payResult   PaidOrder
	payErr      error
}

func (s *fakeOrderService) Create(ctx context.Context, request CreateOrderRequest) (Order, error) {
	return s.createOrder, s.createErr
}

func (s *fakeOrderService) Get(ctx context.Context, orderID string) (Order, error) {
	return s.getOrder, s.getErr
}

func (s *fakeOrderService) Pay(ctx context.Context, orderID string, request PayOrderRequest) (PaidOrder, error) {
	return s.payResult, s.payErr
}

func TestHandlerCreateOrderReturnsCreated(t *testing.T) {
	router := orderTestRouter(&fakeOrderService{
		createOrder: Order{ID: "order-1", Status: "pending", TotalCents: 12999},
	})

	body := bytes.NewBufferString(`{"customer_email":"alex@example.com","items":[{"product_id":"product-1","quantity":1}]}`)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/orders", body)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}

	var payload struct {
		Order Order `json:"order"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Order.ID != "order-1" {
		t.Fatalf("expected order id order-1, got %q", payload.Order.ID)
	}
}

func TestHandlerCreateOrderMapsValidationError(t *testing.T) {
	router := orderTestRouter(&fakeOrderService{createErr: ErrInvalidInput})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"customer_email":"","items":[]}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}

func TestHandlerGetOrderMapsNotFound(t *testing.T) {
	router := orderTestRouter(&fakeOrderService{getErr: ErrNotFound})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/orders/missing-order", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func TestHandlerPayOrderMapsConflict(t *testing.T) {
	router := orderTestRouter(&fakeOrderService{payErr: ErrAlreadyPaid})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/orders/order-1/pay", bytes.NewBufferString(`{"idempotency_key":"payment-1"}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", response.Code)
	}
}

func TestHandlerPayOrderReturnsServerErrorForUnexpectedError(t *testing.T) {
	router := orderTestRouter(&fakeOrderService{payErr: errors.New("database unavailable")})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/orders/order-1/pay", bytes.NewBufferString(`{"idempotency_key":"payment-1"}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}
}

func orderTestRouter(service OrderService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(service).RegisterRoutes(router)
	return router
}
