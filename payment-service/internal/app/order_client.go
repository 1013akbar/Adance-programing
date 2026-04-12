package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type RESTOrderClient struct {
	baseURL string
	client  *http.Client
}

type getOrderResponse struct {
	ID     string `json:"ID"`
	Amount int64  `json:"Amount"`
}

func NewRESTOrderClient(baseURL string, client *http.Client) *RESTOrderClient {
	return &RESTOrderClient{baseURL: baseURL, client: client}
}

func (c *RESTOrderClient) GetOrder(ctx context.Context, orderID string) (int64, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/orders/"+orderID, nil)
	if err != nil {
		return 0, false, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, false, nil
	}
	if resp.StatusCode >= 500 {
		return 0, false, fmt.Errorf("order service returned status %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return 0, false, fmt.Errorf("order lookup failed with status %d", resp.StatusCode)
	}

	var result getOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, false, err
	}

	return result.Amount, true, nil
}
