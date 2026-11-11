package payment

import "time"

type WebhookEvent struct {
	Type      string           `json:"type"`
	EventTime time.Time        `json:"event_time"`
	Data      WebhookEventData `json:"data"`
}

type WebhookEventData struct {
	Order           WebhookOrder           `json:"order"`
	Payment         WebhookPayment         `json:"payment"`
	CustomerDetails WebhookCustomerDetails `json:"customer_details"`
}

type WebhookOrder struct {
	OrderID       string  `json:"order_id"`
	OrderAmount   float64 `json:"order_amount"`
	OrderCurrency string  `json:"order_currency"`
}

type WebhookPayment struct {
	CFPaymentID     string    `json:"cf_payment_id"`
	PaymentStatus   string    `json:"payment_status"`
	PaymentAmount   float64   `json:"payment_amount"`
	PaymentCurrency string    `json:"payment_currency"`
	PaymentMessage  string    `json:"payment_message"`
	PaymentTime     time.Time `json:"payment_time"`
	BankReference   string    `json:"bank_reference"`
	AuthID          string    `json:"auth_id"`
}

type WebhookCustomerDetails struct {
	CustomerID    string `json:"customer_id"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	CustomerPhone string `json:"customer_phone"`
}
