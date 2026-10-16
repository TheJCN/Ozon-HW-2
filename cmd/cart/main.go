package main

import (
	"log"
	"net/http"
	"time"

	pb "OzonHW2/api/product/v1"
	"OzonHW2/internal/cart"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	credentials := insecure.NewCredentials()
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(credentials))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	productClient := pb.NewProductServiceClient(conn)
	service := cart.New(productClient)
	server := &http.Server{}
	server.Addr = ":8080"
	server.Handler = service.Handler()
	server.ReadHeaderTimeout = 5 * time.Second
	log.Println("cart listening on :8080")
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
