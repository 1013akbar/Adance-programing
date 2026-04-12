package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"payment-service/internal/app"
	"payment-service/internal/repository"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	db := connectDB()
	defer db.Close()

	paymentRepo := repository.NewPaymentPostgresRepository(db)
	orderBaseURL := getEnv("ORDER_SERVICE_BASE_URL", "http://localhost:8081")
	orderHTTPClient := &http.Client{Timeout: 2 * time.Second}
	orderClient := app.NewRESTOrderClient(orderBaseURL, orderHTTPClient)
	paymentUC := usecase.NewPaymentService(paymentRepo, orderClient)
	handler := transporthttp.NewHandler(paymentUC)

	router := gin.Default()
	transporthttp.RegisterRoutes(router, handler)

	port := getEnv("HTTP_PORT", "8082")
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func init() {
	fmt.Println("starting payment service")
}
