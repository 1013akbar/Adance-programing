package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"payment-service/internal/app"
	"payment-service/internal/repository"
	transportgrpc "payment-service/internal/transport/grpc"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/taubakabylnurlybek/ap2-generated/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	db := connectDB()
	defer db.Close()

	paymentRepo := repository.NewPaymentPostgresRepository(db)
	orderBaseURL := getEnv("ORDER_SERVICE_BASE_URL", "http://localhost:8081")
	orderHTTPClient := &http.Client{Timeout: 2 * time.Second}
	orderClient := app.NewRESTOrderClient(orderBaseURL, orderHTTPClient)

	// Initialize event publisher (use HTTP Notification Service directly)
	notificationURL := getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083")
	eventPublisher := app.NewHTTPNotificationPublisher(notificationURL, &http.Client{Timeout: 2 * time.Second})

	paymentUC := usecase.NewPaymentService(paymentRepo, orderClient, eventPublisher)
	handler := transporthttp.NewHandler(paymentUC)
	paymentGRPCServer := transportgrpc.NewPaymentServiceServer(paymentUC)

	router := gin.Default()
	transporthttp.RegisterRoutes(router, handler)

	// Start gRPC server
	grpcPort := getEnv("GRPC_PORT", "50051")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor),
	)
	payment.RegisterPaymentServiceServer(grpcServer, paymentGRPCServer)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	port := getEnv("HTTP_PORT", "8082")
	log.Printf("HTTP server listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run payment service: %v", err)
	}
}

func connectDB() *sql.DB {
	dsn := getEnv("DB_DSN", "postgres://payment_user:1234@localhost:5432/payment_db?sslmode=disable")

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

	log.Fatalf("failed to connect payment database: %v", err)
	return nil
}

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)
	log.Printf("gRPC request: method=%s, duration=%v, error=%v", info.FullMethod, duration, err)
	return resp, err
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func init() {
	fmt.Println("starting payment service")
}
