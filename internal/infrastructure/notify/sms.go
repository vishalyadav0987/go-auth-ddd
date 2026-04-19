package notify

import "fmt"

type SMSService interface {
	Send(to, message string) error
}

type DummySMSService struct{}

func NewDummySMSService() *DummySMSService {
	return &DummySMSService{}
}

func (d *DummySMSService) Send(to, message string) error {
	fmt.Println("📱 SMS sent to:", to, "message:", message)
	return nil
}
