package cashfree

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

var (
	ErrMissingWebhookSignature = errors.New("missing webhook signature")
	ErrMissingWebhookTimestamp = errors.New("missing webhook timestamp")
	ErrEmptyWebhookBody        = errors.New("webhook body cannot be empty")
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrMissingClientSecret     = errors.New("cashfree client secret is not configured")
)

// VerifyWebhookSignature verifies that the webhook request
// was signed using our Cashfree client secret.
func (c *Client) VerifyWebhookSignature(
	signature string,
	timestamp string,
	rawBody []byte,
) error {

	if strings.TrimSpace(c.config.ClientSecret) == "" {
		return ErrMissingClientSecret
	}

	if strings.TrimSpace(signature) == "" {
		return ErrMissingWebhookSignature
	}

	if strings.TrimSpace(timestamp) == "" {
		return ErrMissingWebhookTimestamp
	}

	if len(rawBody) == 0 {
		return ErrEmptyWebhookBody
	}

	// Cashfree signs: timestamp + raw request body.
	payload := timestamp + string(rawBody)

	mac := hmac.New(sha256.New, []byte(c.config.ClientSecret))
	mac.Write([]byte(payload))

	expectedSignature := mac.Sum(nil)

	receivedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return ErrInvalidWebhookSignature
	}

	// Constant-time comparison prevents timing-based attacks.
	if !hmac.Equal(expectedSignature, receivedSignature) {
		return ErrInvalidWebhookSignature
	}

	return nil
}
