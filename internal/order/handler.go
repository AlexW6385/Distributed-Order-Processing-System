package order

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router gin.IRoutes) {
	router.POST("/orders", h.create)
	router.GET("/orders/:id", h.get)
	router.POST("/orders/:id/pay", h.pay)
}

type createOrderRequest struct {
	CustomerEmail string                   `json:"customer_email"`
	Items         []createOrderItemRequest `json:"items"`
}

type createOrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type payOrderRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *Handler) create(c *gin.Context) {
	var request createOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	orderRequest := CreateOrderRequest{
		CustomerEmail: request.CustomerEmail,
		Items:         make([]CreateOrderItemRequest, 0, len(request.Items)),
	}

	for _, item := range request.Items {
		orderRequest.Items = append(orderRequest.Items, CreateOrderItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	createdOrder, err := h.service.Create(c.Request.Context(), orderRequest)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order request"})
		case errors.Is(err, ErrInsufficientStock):
			c.JSON(http.StatusConflict, gin.H{"error": "product not found or insufficient stock"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"order": createdOrder})
}

func (h *Handler) get(c *gin.Context) {
	foundOrder, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": foundOrder})
}

func (h *Handler) pay(c *gin.Context) {
	var request payOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	paidOrder, err := h.service.Pay(c.Request.Context(), c.Param("id"), PayOrderRequest{
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment request"})
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		case errors.Is(err, ErrAlreadyPaid):
			c.JSON(http.StatusConflict, gin.H{"error": "order is already paid"})
		case errors.Is(err, ErrCannotBePaid):
			c.JSON(http.StatusConflict, gin.H{"error": "order cannot be paid"})
		case errors.Is(err, ErrConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "payment already exists or idempotency key was used"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to pay order"})
		}
		return
	}

	c.JSON(http.StatusOK, paidOrder)
}
