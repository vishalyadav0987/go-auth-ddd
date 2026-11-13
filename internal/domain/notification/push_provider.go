package notification

import "context"

type PushMessage struct {
	Token string
	Title string
	Body  string
	Data  map[string]string
}

type PushProvider interface {
	Send(
		ctx context.Context,
		message PushMessage,
	) (string, error)
}
