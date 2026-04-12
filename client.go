package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/taubakabylnurlybek/ap2-generated/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run client.go <order_id>")
	}
	orderID := os.Args[1]

	conn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := order.NewOrderServiceClient(conn)

	stream, err := client.SubscribeToOrderUpdates(context.Background(), &order.OrderRequest{OrderId: orderID})
	if err != nil {
		log.Fatalf("failed to subscribe: %v", err)
	}

	log.Printf("Subscribed to updates for order %s", orderID)

	for {
		update, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to receive: %v", err)
		}
		log.Printf("Order Update: ID=%s, Status=%s, Time=%v", update.OrderId, update.Status, update.UpdatedAt.AsTime())
	}
}