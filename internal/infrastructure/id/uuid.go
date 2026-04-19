package id

import "github.com/google/uuid"

type UUIDGenerator struct{}

// Constructor to make accessing the varible from sturct
func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (g *UUIDGenerator) Generate() string {
	return uuid.NewString()
}
