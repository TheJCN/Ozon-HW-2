package main

import (
	"log"
	"net"

	pb "OzonHW2/api/product/v1"
	"OzonHW2/internal/product"
	"google.golang.org/grpc"
)

func main() {
	catalog, err := product.Load("products.json")
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterProductServiceServer(server, catalog)
	log.Println("product-service listening on :50051")
	err = server.Serve(listener)
	if err != nil {
		log.Fatal(err)
	}
}
