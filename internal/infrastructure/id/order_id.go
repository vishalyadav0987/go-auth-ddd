package id

import (
	"github.com/google/uuid"
)

type OrderIDGenerator struct{}

func NewOrderIDGenerator() *OrderIDGenerator {
	return &OrderIDGenerator{}
}

func (g *OrderIDGenerator) NewOrderID() (string, error) {
	return "ORD-" + uuid.NewString(), nil
}
