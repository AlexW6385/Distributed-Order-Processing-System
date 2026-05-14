package order

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestOrderAPIIntegrationCreateGetPay(t *testing.T) {
	db := testutil.OpenTestDB(t)
	productID := testutil.InsertProduct(t, db, "api-keyboard", 12999, 3)

	repository := NewRepository(db)
	service := NewService(repository)
	router := gin.New()
	NewHandler(service).RegisterRoutes(router)

	createResponse := httptest.NewRecorder()
	createBody := bytes.NewBufferString(`{"customer_email":"alex@example.com","items":[{"product_id":"` + productID + `","quantity":1}]}`)
	createRequest := httptest.NewRequest(http.MethodPost, "/orders", createBody)
	createRequest.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var createPayload struct {
		Order Order `json:"order"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	getResponse := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/orders/"+createPayload.Order.ID, nil)

	router.ServeHTTP(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d: %s", getResponse.Code, getResponse.Body.String())
	}

	payResponse := httptest.NewRecorder()
	payRequest := httptest.NewRequest(http.MethodPost, "/orders/"+createPayload.Order.ID+"/pay", bytes.NewBufferString(`{"idempotency_key":"api-payment-1"}`))
	payRequest.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(payResponse, payRequest)

	if payResponse.Code != http.StatusOK {
		t.Fatalf("expected pay status 200, got %d: %s", payResponse.Code, payResponse.Body.String())
	}

	var payPayload PaidOrder
	if err := json.Unmarshal(payResponse.Body.Bytes(), &payPayload); err != nil {
		t.Fatalf("decode pay response: %v", err)
	}
	if payPayload.Order.Status != "paid" {
		t.Fatalf("expected paid order, got %q", payPayload.Order.Status)
	}
}
