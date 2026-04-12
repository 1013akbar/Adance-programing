package main

import (
	"database/sql"
	"log"
	"net/http"
	"order-service/internal/app"
	"order-service/internal/repository"
	transport "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	db := connectDB()
	defer db.Close()

	paymentBaseURL := getEnv("PAYMENT_SERVICE_BASE_URL", "http://localhost:8082")
	paymentHTTPClient := &http.Client{Timeout: 2 * time.Second}
	paymentClient := app.NewRESTPaymentClient(paymentBaseURL, paymentHTTPClient)

	orderRepo := repository.NewOrderPostgresRepository(db)
	orderUC := usecase.NewOrderService(orderRepo, paymentClient)
	handler := transport.NewHandler(orderUC)

	router := gin.Default()
	transport.RegisterRoutes(router, handler)

	port := getEnv("HTTP_PORT", "8081")
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
