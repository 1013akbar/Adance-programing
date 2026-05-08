package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type PaymentEvent struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type NotificationMessage struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
	Timestamp     string `json:"timestamp"`
}

var (
	notificationsMu sync.Mutex
	notifications   []NotificationMessage
)

type incomingNotification struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
	Timestamp     string `json:"timestamp"`
}

func main() {
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	queueName := getEnv("QUEUE_NAME", "payment.completed")

	var conn *amqp091.Connection
	var err error
	conn, err = amqp091.Dial(rabbitURL)
	if err != nil {
		log.Printf("RabbitMQ unavailable, starting notification HTTP API without queue consumer: %v", err)
	} else {
		defer conn.Close()

		ch, err := conn.Channel()
		if err != nil {
			log.Printf("failed to open RabbitMQ channel, continuing without consumer: %v", err)
		} else {
			defer ch.Close()

			q, err := ch.QueueDeclare(
				queueName,
				true,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				log.Printf("failed to declare queue, continuing without consumer: %v", err)
			} else {
				err = ch.Qos(1, 0, false)
				if err != nil {
					log.Printf("failed to set QoS, continuing without consumer: %v", err)
				} else {
					msgs, err := ch.Consume(
						q.Name,
						"",
						false,
						false,
						false,
						false,
						nil,
					)
					if err != nil {
						log.Printf("failed to register consumer, continuing without consumer: %v", err)
					} else {
						log.Printf("Notification service started and listening on queue %s", queueName)
						go consumeMessages(msgs)
					}
				}
			}
		}
	}
	log.Println("Notification service HTTP API is starting")

	httpPort := getEnv("HTTP_PORT", "8083")
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	http.HandleFunc("/notifications", notificationsHandler)
	log.Printf("HTTP endpoints listening on :%s", httpPort)
	go func() {
		if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
			log.Printf("notification HTTP server stopped: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down notification service...")
}

func consumeMessages(msgs <-chan amqp091.Delivery) {
	for d := range msgs {
		processMessage(d)
	}
}

func processMessage(d amqp091.Delivery) {
	var event PaymentEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		d.Nack(false, false)
		return
	}

	notification := NotificationMessage{
		OrderID:       event.OrderID,
		Amount:        event.Amount,
		CustomerEmail: event.CustomerEmail,
		Status:        event.Status,
		Timestamp:     time.Now().Format(time.RFC3339),
	}
	storeNotification(notification)

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f",
		event.CustomerEmail, event.OrderID, float64(event.Amount)/100)

	d.Ack(false)
}

func storeNotification(message NotificationMessage) {
	notificationsMu.Lock()
	defer notificationsMu.Unlock()
	notifications = append(notifications, message)
	if len(notifications) > 100 {
		notifications = notifications[len(notifications)-100:]
	}
}

func notificationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var incoming incomingNotification
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, "invalid notification payload", http.StatusBadRequest)
			return
		}

		if incoming.Timestamp == "" {
			incoming.Timestamp = time.Now().Format(time.RFC3339)
		}

		storeNotification(NotificationMessage{
			OrderID:       incoming.OrderID,
			Amount:        incoming.Amount,
			CustomerEmail: incoming.CustomerEmail,
			Status:        incoming.Status,
			Timestamp:     incoming.Timestamp,
		})
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("accepted"))
		return
	}

	notificationsMu.Lock()
	defer notificationsMu.Unlock()

	response := make([]NotificationMessage, len(notifications))
	copy(response, notifications)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode notifications: %v", err)
		http.Error(w, "failed to return notifications", http.StatusInternalServerError)
	}
}

func init() {
	fmt.Println("starting notification service")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
