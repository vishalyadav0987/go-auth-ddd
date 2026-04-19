package authapp

type EmailService interface {
	Send(to, message string) error
}

type SMSService interface {
	Send(to, message string) error
}
