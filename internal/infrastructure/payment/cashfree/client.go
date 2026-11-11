package cashfree

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	APIVersion   string
}

type Client struct {
	config     Config
	httpClient *http.Client
}

func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://sandbox.cashfree.com/pg"
	}

	if config.APIVersion == "" {
		config.APIVersion = "2023-08-01"
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type CreateOrderRequest struct {
	OrderID         string          `json:"order_id"`
	OrderAmount     string          `json:"order_amount"`
	OrderCurrency   string          `json:"order_currency"`
	CustomerDetails CustomerDetails `json:"customer_details"`
	OrderMeta       OrderMeta       `json:"order_meta,omitempty"`
}

type CustomerDetails struct {
	CustomerID    string `json:"customer_id"`
	CustomerPhone string `json:"customer_phone"`
	CustomerEmail string `json:"customer_email"`
}

type OrderMeta struct {
	ReturnURL string `json:"return_url,omitempty"`
	NotifyURL string `json:"notify_url,omitempty"`
}

type CreateOrderResponse struct {
	CFOrderID        string `json:"cf_order_id"`
	OrderID          string `json:"order_id"`
	OrderStatus      string `json:"order_status"`
	PaymentSessionID string `json:"payment_session_id"`
}

type ErrorResponse struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) CreateOrder(
	ctx context.Context,
	idempotencyKey string,
	input CreateOrderRequest,
) (*CreateOrderResponse, error) {
	if strings.TrimSpace(c.config.ClientID) == "" {
		return nil, fmt.Errorf("cashfree client ID is required")
	}

	if strings.TrimSpace(c.config.ClientSecret) == "" {
		return nil, fmt.Errorf("cashfree client secret is required")
	}

	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, fmt.Errorf("cashfree idempotency key is required")
	}

	if strings.TrimSpace(input.OrderID) == "" {
		return nil, fmt.Errorf("cashfree order ID is required")
	}

	if input.OrderAmount == "" || input.OrderAmount == "0" {
		return nil, fmt.Errorf("cashfree order amount is required")
	}

	if input.OrderCurrency == "" {
		input.OrderCurrency = "INR"
	}

	requestBody, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal cashfree request: %w", err)
	}

	url := strings.TrimRight(c.config.BaseURL, "/") + "/orders"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create cashfree request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-client-id", c.config.ClientID)
	req.Header.Set("x-client-secret", c.config.ClientSecret)
	req.Header.Set("x-api-version", c.config.APIVersion)
	req.Header.Set("x-idempotency-key", idempotencyKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send cashfree request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read cashfree response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var cashfreeError ErrorResponse

		if err := json.Unmarshal(responseBody, &cashfreeError); err != nil {
			return nil, fmt.Errorf(
				"cashfree API returned status %d: %s",
				resp.StatusCode,
				string(responseBody),
			)
		}

		return nil, fmt.Errorf(
			"cashfree API error (%s): %s",
			cashfreeError.Code,
			cashfreeError.Message,
		)
	}

	var result CreateOrderResponse

	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode cashfree response: %w", err)
	}

	if result.OrderID == "" || result.PaymentSessionID == "" {
		return nil, fmt.Errorf("cashfree response missing order ID or payment session ID")
	}

	return &result, nil
}
