package order

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/events"
	"github.com/google/uuid"
)

type Service struct {
	repository    Store
	productClient ProductClient
	paymentClient PaymentClient
}

type Store interface {
	Create(ctx context.Context, request CreateOrderRequest, reservedItems []ReservedStock) (Order, error)
	Find(ctx context.Context, orderID string) (Order, error)
	MarkPaid(ctx context.Context, orderID string, event events.OrderPaidEvent) (Order, error)
}

type ProductClient interface {
	ReserveStock(ctx context.Context, productID string, quantity int) (ReservedStock, error)
}

type PaymentClient interface {
	PayOrder(ctx context.Context, orderID string, idempotencyKey string, amountCents int) (Payment, error)
}

type Option func(*Service)

func WithProductClient(client ProductClient) Option {
	return func(service *Service) {
		service.productClient = client
	}
}

func WithPaymentClient(client PaymentClient) Option {
	return func(service *Service) {
		service.paymentClient = client
	}
}

func NewService(repository Store, options ...Option) *Service {
	service := &Service{repository: repository}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *Service) Create(ctx context.Context, request CreateOrderRequest) (Order, error) {
	request.CustomerEmail = strings.TrimSpace(request.CustomerEmail)
	if request.CustomerEmail == "" || len(request.Items) == 0 {
		return Order{}, ErrInvalidInput
	}

	for i := range request.Items {
		request.Items[i].ProductID = strings.TrimSpace(request.Items[i].ProductID)
		if request.Items[i].ProductID == "" || request.Items[i].Quantity <= 0 {
			return Order{}, ErrInvalidInput
		}
	}

	reservedItems := make([]ReservedStock, 0, len(request.Items))
	for _, item := range request.Items {
		reservedStock, err := s.productClient.ReserveStock(ctx, item.ProductID, item.Quantity)
		if err != nil {
			return Order{}, err
		}
		reservedItems = append(reservedItems, reservedStock)
	}

	return s.repository.Create(ctx, request, reservedItems)
}

func (s *Service) Get(ctx context.Context, orderID string) (Order, error) {
	foundOrder, err := s.repository.Find(ctx, strings.TrimSpace(orderID))
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	return foundOrder, err
}

func (s *Service) Pay(ctx context.Context, orderID string, request PayOrderRequest) (PaidOrder, error) {
	orderID = strings.TrimSpace(orderID)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if orderID == "" || request.IdempotencyKey == "" {
		return PaidOrder{}, ErrInvalidInput
	}

	foundOrder, err := s.Get(ctx, orderID)
	if err != nil {
		return PaidOrder{}, err
	}

	if foundOrder.Status == "paid" {
		return PaidOrder{}, ErrAlreadyPaid
	}

	if foundOrder.Status != "pending" {
		return PaidOrder{}, ErrCannotBePaid
	}

	createdPayment, err := s.paymentClient.PayOrder(ctx, foundOrder.ID, request.IdempotencyKey, foundOrder.TotalCents)
	if err != nil {
		return PaidOrder{}, err
	}

	orderPaidEvent := events.OrderPaidEvent{
		EventID:       uuid.NewString(),
		OrderID:       foundOrder.ID,
		PaymentID:     createdPayment.ID,
		CustomerEmail: foundOrder.CustomerEmail,
		AmountCents:   foundOrder.TotalCents,
		PaidAt:        createdPayment.CreatedAt,
	}

	paidOrder, err := s.repository.MarkPaid(ctx, foundOrder.ID, orderPaidEvent)
	if err != nil {
		return PaidOrder{}, err
	}

	return PaidOrder{
		Order:   paidOrder,
		Payment: createdPayment,
	}, nil
}
