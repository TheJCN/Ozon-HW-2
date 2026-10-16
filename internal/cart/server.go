package cart

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	pb "OzonHW2/api/product/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Item struct {
	SKU   int64  `json:"sku"`
	Name  string `json:"name"`
	Count uint32 `json:"count"`
	Price uint32 `json:"price"`
}

type AddRequest struct {
	Count uint32 `json:"count"`
}

type Cart struct {
	Items      []Item `json:"items"`
	TotalPrice uint32 `json:"total_price"`
}

type Server struct {
	products pb.ProductServiceClient
	carts    map[int64]map[int64]Item
}

func New(products pb.ProductServiceClient) *Server {
	server := &Server{}
	server.products = products
	server.carts = make(map[int64]map[int64]Item)
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /user/{user_id}/cart/{sku_id}", s.add)
	mux.HandleFunc("GET /user/{user_id}/cart", s.get)
	return mux
}

func (s *Server) add(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.PathValue("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		http.Error(w, "user_id must be positive", http.StatusBadRequest)
		return
	}

	sku, err := strconv.ParseInt(r.PathValue("sku_id"), 10, 64)
	if err != nil || sku <= 0 {
		http.Error(w, "sku_id must be positive", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
	if err != nil {
		http.Error(w, "cannot read request body", http.StatusBadRequest)
		return
	}

	var request AddRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		http.Error(w, "invalid JSON or count", http.StatusBadRequest)
		return
	}
	if request.Count == 0 {
		http.Error(w, "count must be positive", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	productRequest := &pb.GetProductRequest{Sku: sku}
	product, err := s.products.GetProduct(ctx, productRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			http.Error(w, "product not found", http.StatusPreconditionFailed)
		} else {
			http.Error(w, "product service unavailable", http.StatusInternalServerError)
		}
		return
	}
	if product.Price < 0 {
		http.Error(w, "invalid product price", http.StatusInternalServerError)
		return
	}

	items := s.carts[userID]
	if items == nil {
		items = make(map[int64]Item)
	}
	item := items[sku]
	if request.Count > math.MaxUint32-item.Count {
		http.Error(w, "count overflow", http.StatusBadRequest)
		return
	}
	item.SKU = sku
	item.Name = product.Name
	item.Price = uint32(product.Price)
	item.Count = item.Count + request.Count
	items[sku] = item
	s.carts[userID] = items
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.PathValue("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		http.Error(w, "user_id must be positive", http.StatusBadRequest)
		return
	}

	var result Cart
	for _, item := range s.carts[userID] {
		result.Items = append(result.Items, item)
	}
	if len(result.Items) == 0 {
		http.Error(w, "cart is empty", http.StatusNotFound)
		return
	}
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].SKU < result.Items[j].SKU
	})
	var total uint64
	for _, item := range result.Items {
		total += uint64(item.Price) * uint64(item.Count)
		if total > math.MaxUint32 {
			http.Error(w, "total price overflow", http.StatusInternalServerError)
			return
		}
	}
	result.TotalPrice = uint32(total)
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(result)
}
