package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HTTPNotificationPublisher struct {
	baseURL string
	client  *http.Client
}

type notificationPayload struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
	Timestamp     string `json:"timestamp"`
}

func NewHTTPNotificationPublisher(baseURL string, client *http.Client) *HTTPNotificationPublisher {
	return &HTTPNotificationPublisher{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (p *HTTPNotificationPublisher) PublishPaymentEvent(ctx context.Context, orderID string, amount int64, status string) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("notification publisher is not available")
	}

	payload := notificationPayload{
		OrderID:       orderID,
		Amount:        amount,
		CustomerEmail: fmt.Sprintf("%s@example.com", orderID),
		Status:        status,
		Timestamp:     time.Now().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/notifications", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	return nil
}
