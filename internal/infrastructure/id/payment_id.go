package id

import (
	"github.com/google/uuid"
)

type PaymentIDGenerator struct{}

func NewPaymentIDGenerator() *PaymentIDGenerator {
	return &PaymentIDGenerator{}
}

func (g *PaymentIDGenerator) NewPaymentID() (string, error) {
	return "PAY-" + uuid.NewString(), nil
}

func (g *PaymentIDGenerator) NewOrderID() string {
	return "PAY-" + uuid.NewString()
}
