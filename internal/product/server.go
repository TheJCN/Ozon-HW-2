package product

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"

	pb "OzonHW2/api/product/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Product struct {
	Name  string `json:"name"`
	Price uint32 `json:"price"`
}

type Server struct {
	pb.UnimplementedProductServiceServer
	products map[int64]Product
}

func Load(path string) (*Server, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records map[string]Product
	err = json.Unmarshal(data, &records)
	if err != nil {
		return nil, err
	}
	products := make(map[int64]Product)
	for key, item := range records {
		sku, err := strconv.ParseInt(key, 10, 64)
		if err != nil || sku <= 0 {
			return nil, fmt.Errorf("invalid SKU: %q", key)
		}
		if item.Price > math.MaxInt32 {
			return nil, fmt.Errorf("price exceeds int32 for SKU %d", sku)
		}
		products[sku] = item
	}
	server := &Server{}
	server.products = products
	return server, nil
}

func (s *Server) GetProduct(_ context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	if req.GetSku() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "sku must be positive")
	}

	item, exists := s.products[req.Sku]
	if !exists {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	response := &pb.GetProductResponse{}
	response.Sku = req.Sku
	response.Name = item.Name
	response.Price = int32(item.Price)
	return response, nil
}
