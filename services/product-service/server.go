package main

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"time"

	productv1 "github.com/AlexW6385/Distributed-Order-Processing-System/gen/product/v1"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/product"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type productServer struct {
	productv1.UnimplementedProductServiceServer
	service *product.Service
}

func newProductServer(service *product.Service) *productServer {
	return &productServer{service: service}
}

func (s *productServer) ListProducts(ctx context.Context, request *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	products, err := s.service.List(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list products")
	}

	response := &productv1.ListProductsResponse{
		Products: make([]*productv1.Product, 0, len(products)),
	}

	for _, item := range products {
		response.Products = append(response.Products, toProtoProduct(item))
	}

	return response, nil
}

func (s *productServer) ReserveStock(ctx context.Context, request *productv1.ReserveStockRequest) (*productv1.ReserveStockResponse, error) {
	if request.GetProductId() == "" || request.GetQuantity() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id and positive quantity are required")
	}

	reservation, err := s.service.ReserveStock(ctx, request.GetProductId(), int(request.GetQuantity()))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.FailedPrecondition, "product not found or insufficient stock")
		}
		return nil, status.Error(codes.Internal, "failed to reserve stock")
	}

	return &productv1.ReserveStockResponse{
		ProductId:      reservation.ProductID,
		Quantity:       int32(reservation.Quantity),
		UnitPriceCents: int32(reservation.UnitPriceCents),
		SubtotalCents:  int32(reservation.SubtotalCents),
	}, nil
}

func listenGRPC(port string) (net.Listener, error) {
	return net.Listen("tcp", ":"+port)
}

func toProtoProduct(product product.Product) *productv1.Product {
	return &productv1.Product{
		Id:            product.ID,
		Sku:           product.SKU,
		Name:          product.Name,
		Description:   product.Description,
		PriceCents:    int32(product.PriceCents),
		StockQuantity: int32(product.StockQuantity),
		CreatedAt:     product.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:     product.UpdatedAt.Format(time.RFC3339Nano),
	}
}
