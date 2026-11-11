package payment

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	domainpayment "github.com/vishalyadav0987/authentication/internal/domain/payment"
)

type HandleWebhookUseCase struct {
	paymentRepository domainpayment.PaymentRepository
}

func NewHandleWebhookUseCase(
	paymentRepository domainpayment.PaymentRepository,
) *HandleWebhookUseCase {
	return &HandleWebhookUseCase{
		paymentRepository: paymentRepository,
	}
}

func (uc *HandleWebhookUseCase) Execute(
	ctx context.Context,
	event domainpayment.WebhookEvent,
) error {

	// 1. Validate webhook information.
	if event.Data.Order.OrderID == "" {
		return errors.New("cashfree order ID is required")
	}

	if event.Data.Payment.PaymentStatus == "" {
		return errors.New("payment status is required")
	}

	if event.Data.Order.OrderCurrency != "INR" ||
		event.Data.Payment.PaymentCurrency != "INR" {
		return errors.New("invalid payment currency")
	}

	// 2. Find payment using Cashfree merchant order ID.
	payment, err := uc.paymentRepository.GetByCashFreeOrderID(
		ctx,
		event.Data.Order.OrderID,
	)
	if err != nil {
		return err
	}

	// 3. Convert amounts into paise.
	paymentAmountPaise := int64(math.Round(
		event.Data.Payment.PaymentAmount * 100,
	))

	orderAmountPaise := int64(math.Round(
		event.Data.Order.OrderAmount * 100,
	))

	// 4. Validate both Cashfree amounts.
	if paymentAmountPaise != payment.AmountPaise ||
		orderAmountPaise != payment.AmountPaise {
		return domainpayment.ErrPaymentAmountMismatch
	}

	// 5. Map Cashfree status to our domain status.
	var status domainpayment.PaymentStatus

	switch event.Data.Payment.PaymentStatus {
	case "SUCCESS":
		status = domainpayment.PaymentStatusSuccess

	case "FAILED":
		status = domainpayment.PaymentStatusFailed

	case "USER_DROPPED":
		status = domainpayment.PaymentStatusCancelled

	case "PENDING":
		status = domainpayment.PaymentStatusPending

	default:
		return errors.New("unsupported cashfree payment status")
	}

	// 6. Get actual Cashfree payment ID.
	// cashfreePaymentID := strings.TrimSpace(),

	if event.Data.Payment.CFPaymentID == "0" {
		return errors.New("invalid cashfree payment ID")
	}

	// 7. Prepare settlement information.
	settlement := &domainpayment.WebhookSettlement{
		PaymentID:         payment.PaymentID,
		CashfreePaymentID: event.Data.Payment.CFPaymentID,
		AmountPaise:       paymentAmountPaise,
		Currency:          event.Data.Payment.PaymentCurrency,
		Status:            status,
	}

	// 8. Create an outbox event only for successful payments.
	if status == domainpayment.PaymentStatusSuccess {

		settlement.EventID = "payment-succeeded:" + payment.PaymentID
		settlement.EventType = "PAYMENT_SUCCEEDED"

		payload := struct {
			PaymentID         string `json:"payment_id"`
			OrderID           string `json:"order_id"`
			CashfreePaymentID string `json:"cashfree_payment_id"`
			AmountPaise       int64  `json:"amount_paise"`
			Currency          string `json:"currency"`
			Status            string `json:"status"`
		}{
			PaymentID:         payment.PaymentID,
			OrderID:           payment.OrderID,
			CashfreePaymentID: event.Data.Payment.CFPaymentID,
			AmountPaise:       paymentAmountPaise,
			Currency:          "INR",
			Status:            string(status),
		}

		settlement.EventPayload, err = json.Marshal(payload)
		if err != nil {
			return err
		}

		settlement.EventCreatedAt = event.EventTime

		if settlement.EventCreatedAt.IsZero() {
			settlement.EventCreatedAt = time.Now().UTC()
		}
	}

	// 9. Apply payment, order and outbox changes atomically.
	return uc.paymentRepository.ApplyWebhook(
		ctx,
		settlement,
	)
}
