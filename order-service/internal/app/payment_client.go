package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
)

type RESTPaymentClient struct {
	baseURL string
	client  *http.Client
}

type authorizePaymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type authorizePaymentResponse struct {
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id"`
}

type getPaymentResponse struct {
	Status string `json:"status"`
}

func NewRESTPaymentClient(baseURL string, client *http.Client) *RESTPaymentClient {
	return &RESTPaymentClient{baseURL: baseURL, client: client}
}

func (c *RESTPaymentClient) AuthorizePayment(ctx context.Context, orderID string, amount int64) (string, string, error) {
	payload := authorizePaymentRequest{OrderID: orderID, Amount: amount}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "", "", err
		}
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", "", fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("payment authorization failed with status %d", resp.StatusCode)
	}

	var result authorizePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return result.Status, result.TransactionID, nil
}

func (c *RESTPaymentClient) GetPaymentStatus(ctx context.Context, orderID string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/payments/"+orderID, nil)
	if err != nil {
		return "", false, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "", false, err
		}
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode >= 500 {
		return "", false, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return "", false, fmt.Errorf("payment lookup failed with status %d", resp.StatusCode)
	}

	var result getPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false, err
	}

	return result.Status, true, nil
}
