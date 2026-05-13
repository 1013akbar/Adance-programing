package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"notification-service/internal/notification"
	"time"

	"github.com/go-redis/redis/v8"
)

// NotificationJob represents a background job for sending notifications
type NotificationJob struct {
	PaymentID     string `json:"payment_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
	RetryCount    int    `json:"retry_count"`
}

// BackgroundWorker handles notification jobs with retry logic
type BackgroundWorker struct {
	redisClient *redis.Client
	emailSender notification.EmailSender
	maxRetries  int
	baseDelay   time.Duration
}

func NewBackgroundWorker(redisClient *redis.Client, emailSender notification.EmailSender, maxRetries int, baseDelay time.Duration) *BackgroundWorker {
	return &BackgroundWorker{
		redisClient: redisClient,
		emailSender: emailSender,
		maxRetries:  maxRetries,
		baseDelay:   baseDelay,
	}
}

// ProcessJob processes a notification job with idempotency and retry logic
func (w *BackgroundWorker) ProcessJob(ctx context.Context, job NotificationJob) error {
	// Check idempotency using Redis
	idempotencyKey := fmt.Sprintf("notification:%s", job.PaymentID)
	processed, err := w.redisClient.Get(ctx, idempotencyKey).Result()
	if err == nil && processed == "processed" {
		log.Printf("Notification for payment %s already processed, skipping", job.PaymentID)
		return nil
	}

	// Process the notification
	err = w.sendNotification(ctx, job)
	if err != nil {
		job.RetryCount++
		if job.RetryCount < w.maxRetries {
			// Schedule retry with exponential backoff
			delay := w.baseDelay * time.Duration(1<<uint(job.RetryCount-1)) // 2^(retry-1) * baseDelay
			log.Printf("Notification failed, retrying in %v (attempt %d/%d)", delay, job.RetryCount, w.maxRetries)

			// Store job for retry
			jobJSON, _ := json.Marshal(job)
			w.redisClient.Set(ctx, fmt.Sprintf("retry:%s", job.PaymentID), jobJSON, delay+time.Minute)

			return fmt.Errorf("job scheduled for retry: %v", err)
		} else {
			log.Printf("Notification failed permanently after %d retries: %v", w.maxRetries, err)
			return err
		}
	}

	// Mark as processed
	w.redisClient.Set(ctx, idempotencyKey, "processed", 24*time.Hour)
	log.Printf("Notification sent successfully for payment %s", job.PaymentID)
	return nil
}

// sendNotification sends the actual notification
func (w *BackgroundWorker) sendNotification(ctx context.Context, job NotificationJob) error {
	subject := fmt.Sprintf("Payment %s - Order #%s", job.Status, job.OrderID)
	body := fmt.Sprintf("Your payment of $%.2f for order #%s has been %s.",
		float64(job.Amount)/100, job.OrderID, job.Status)

	return w.emailSender.SendEmail(ctx, job.CustomerEmail, subject, body)
}

// StartRetryProcessor starts a goroutine to process retry jobs
func (w *BackgroundWorker) StartRetryProcessor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.processRetries(ctx)
			}
		}
	}()
}

// processRetries checks for and processes retry jobs
func (w *BackgroundWorker) processRetries(ctx context.Context) {
	keys, err := w.redisClient.Keys(ctx, "retry:*").Result()
	if err != nil {
		log.Printf("Error checking retry keys: %v", err)
		return
	}

	for _, key := range keys {
		jobJSON, err := w.redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var job NotificationJob
		if json.Unmarshal([]byte(jobJSON), &job) != nil {
			continue
		}

		// Process the retry job
		if err := w.ProcessJob(ctx, job); err == nil {
			// Remove from retry queue on success
			w.redisClient.Del(ctx, key)
		}
	}
}
