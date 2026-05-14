package product

import (
	"context"
	"time"

	productv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/product/v1"
)

type GRPCClient struct {
	client productv1.ProductServiceClient
}

func NewGRPCClient(client productv1.ProductServiceClient) *GRPCClient {
	return &GRPCClient{client: client}
}

func (c *GRPCClient) List(ctx context.Context) ([]Product, error) {
	response, err := c.client.ListProducts(ctx, &productv1.ListProductsRequest{})
	if err != nil {
		return nil, err
	}

	products := make([]Product, 0, len(response.GetProducts()))
	for _, item := range response.GetProducts() {
		products = append(products, fromProtoProduct(item))
	}

	return products, nil
}

func (c *GRPCClient) ReserveStock(ctx context.Context, productID string, quantity int) (ReservedStock, error) {
	response, err := c.client.ReserveStock(ctx, &productv1.ReserveStockRequest{
		ProductId: productID,
		Quantity:  int32(quantity),
	})
	if err != nil {
		return ReservedStock{}, err
	}

	return ReservedStock{
		ProductID:      response.GetProductId(),
		Quantity:       int(response.GetQuantity()),
		UnitPriceCents: int(response.GetUnitPriceCents()),
		SubtotalCents:  int(response.GetSubtotalCents()),
	}, nil
}

func fromProtoProduct(item *productv1.Product) Product {
	createdAt, _ := time.Parse(time.RFC3339Nano, item.GetCreatedAt())
	updatedAt, _ := time.Parse(time.RFC3339Nano, item.GetUpdatedAt())

	return Product{
		ID:            item.GetId(),
		SKU:           item.GetSku(),
		Name:          item.GetName(),
		Description:   item.GetDescription(),
		PriceCents:    int(item.GetPriceCents()),
		StockQuantity: int(item.GetStockQuantity()),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
