package order

import (
	"context"

	productv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductGRPCClient struct {
	client productv1.ProductServiceClient
}

func NewProductGRPCClient(client productv1.ProductServiceClient) *ProductGRPCClient {
	return &ProductGRPCClient{client: client}
}

func (c *ProductGRPCClient) ReserveStock(ctx context.Context, productID string, quantity int) (ReservedStock, error) {
	response, err := c.client.ReserveStock(ctx, &productv1.ReserveStockRequest{
		ProductId: productID,
		Quantity:  int32(quantity),
	})
	if err != nil {
		if status.Code(err) == codes.FailedPrecondition {
			return ReservedStock{}, ErrInsufficientStock
		}
		return ReservedStock{}, err
	}

	return ReservedStock{
		ProductID:      response.GetProductId(),
		Quantity:       int(response.GetQuantity()),
		UnitPriceCents: int(response.GetUnitPriceCents()),
		SubtotalCents:  int(response.GetSubtotalCents()),
	}, nil
}
