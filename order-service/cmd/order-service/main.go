package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"order-service/internal/app"
	"order-service/internal/repository"
	transportgrpc "order-service/internal/transport/grpc"
	transport "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/taubakabylnurlybek/ap2-generated/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	db := connectDB()
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Initialize Redis client
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Redis unavailable, continuing without cache: %v", err)
		redisClient = nil
	}

	paymentServiceAddr := getEnv("PAYMENT_SERVICE_ADDR", "localhost:50051")
	conn, err := grpc.Dial(paymentServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to payment service: %v", err)
	}
	defer conn.Close()

	paymentClient := app.NewGRPCPaymentClient(conn)

	orderRepo := repository.NewOrderPostgresRepository(db)
	orderUC := usecase.NewOrderService(orderRepo, paymentClient, redisClient)
	orderGRPCServer := transportgrpc.NewOrderServiceServer(orderUC)
	orderUC.SetStatusUpdater(orderGRPCServer)
	handler := transport.NewHandler(orderUC)

	// Start gRPC server
	grpcPort := getEnv("GRPC_PORT", "50052")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	order.RegisterOrderServiceServer(grpcServer, orderGRPCServer)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	router := gin.Default()
	transport.RegisterRoutes(router, handler, redisClient)

	port := getEnv("HTTP_PORT", "8081")
	log.Printf("HTTP server listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run order service: %v", err)
	}
}

func connectDB() *sql.DB {
	dsn := getEnv("DB_DSN", "postgres://order_user:1234@localhost:5432/order_db?sslmode=disable")

	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			return db
		}
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("failed to connect order database: %v", err)
	return nil
}

func runMigrations(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS orders (
		    id TEXT PRIMARY KEY,
		    customer_id TEXT NOT NULL,
		    item_name TEXT NOT NULL,
		    amount BIGINT NOT NULL CHECK (amount > 0),
		    status TEXT NOT NULL CHECK (status IN ('Pending', 'Paid', 'Failed', 'Cancelled')),
		    created_at TIMESTAMP NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS order_idempotency (
		    idempotency_key TEXT PRIMARY KEY,
		    order_id TEXT NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
