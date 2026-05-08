package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type HTTPEventPublisher struct {
	notificationServiceURL string
	httpClient             *http.Client
}

func NewHTTPEventPublisher(notificationServiceURL string) *HTTPEventPublisher {
	return &HTTPEventPublisher{
		notificationServiceURL: notificationServiceURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (p *HTTPEventPublisher) PublishPaymentEvent(ctx context.Context, orderID string, amount int64, status string) error {
	event := PaymentEvent{
		OrderID:       orderID,
		Amount:        amount,
		CustomerEmail: fmt.Sprintf("%s@example.com", orderID),
		Status:        status,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.notificationServiceURL+"/events", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to send payment event via HTTP: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("Notification service returned status %d", resp.StatusCode)
		return fmt.Errorf("notification service error: status %d", resp.StatusCode)
	}

	log.Printf("Published payment event for order %s via HTTP", orderID)
	return nil
}
