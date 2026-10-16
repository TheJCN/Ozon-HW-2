package cart

import (
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pb "OzonHW2/api/product/v1"
	"OzonHW2/internal/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestServices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "products.json")
	if err := os.WriteFile(path, []byte(`{"1":{"name":"one","price":10},"2":{"name":"two","price":20}}`), 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := product.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	pb.RegisterProductServiceServer(server, catalog)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	client := pb.NewProductServiceClient(conn)
	for _, tc := range []struct {
		sku  int64
		code codes.Code
	}{{0, codes.InvalidArgument}, {-1, codes.InvalidArgument}, {99, codes.NotFound}, {1, codes.OK}} {
		_, err := client.GetProduct(context.Background(), &pb.GetProductRequest{Sku: tc.sku})
		if status.Code(err) != tc.code {
			t.Fatalf("sku %d: %v", tc.sku, err)
		}
	}
	handler := New(client).Handler()
	request := func(method, path, body string, code int) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		if rec.Code != code {
			t.Errorf("%s %s: got %d want %d: %s", method, path, rec.Code, code, rec.Body.String())
		}
		return rec
	}
	request("GET", "/user/1/cart", "", 404)
	request("GET", "/user/0/cart", "", 400)
	for _, path := range []string{"/user/0/cart/1", "/user/-1/cart/1", "/user/1/cart/0", "/user/1/cart/-1", "/user/abc/cart/1"} {
		request("POST", path, `{"count":1}`, 400)
	}
	for _, body := range []string{`{}`, `{"count":0}`, `{"count":-1}`, `{"count":4294967296}`, `{"count":1.5}`, `broken`, `{"count":1} {}`} {
		request("POST", "/user/1/cart/1", body, 400)
	}
	request("POST", "/user/1/cart/99", `{"count":1}`, 412)
	request("POST", "/user/1/cart/2", `{"count":1}`, 204)
	request("POST", "/user/1/cart/1", `{"count":2}`, 204)
	request("POST", "/user/1/cart/1", `{"count":3}`, 204)
	rec := request("GET", "/user/1/cart", "", 200)
	var result Cart
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 || result.Items[0].SKU != 1 || result.Items[0].Count != 5 || result.Items[0].Name != "one" || result.TotalPrice != 70 {
		t.Fatalf("unexpected cart: %+v", result)
	}
	request("GET", "/user/2/cart", "", 404)
	request("POST", "/user/4/cart/1", `{"count":4294967295}`, 204)
	request("POST", "/user/4/cart/1", `{"count":1}`, 400)
	request("GET", "/user/4/cart", "", 500)
	conn.Close()
	request("POST", "/user/5/cart/1", `{"count":1}`, 500)
	request("GET", "/user/5/cart", "", 404)
	// The cart retains product details even when the product service is unavailable.
	request("GET", "/user/1/cart", "", 200)
}
