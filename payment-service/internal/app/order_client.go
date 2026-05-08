package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return 0, false, nil
	}
	if resp.StatusCode >= 400 {
		errMsg := strings.TrimSpace(string(body))
		if errMsg == "" {
			errMsg = fmt.Sprintf("order service returned status %d", resp.StatusCode)
		}
		return 0, false, fmt.Errorf(errMsg)
	}

	var result getOrderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, false, err
	}

	return result.Amount, true, nil
}
