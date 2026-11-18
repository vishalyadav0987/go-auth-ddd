// Package notiflog is the structured logger for the notification flow
// only (send → outbox → queue → delivery → retry / DLQ).
//
// Every record's message is a stable event name such as
// "delivery.retry_scheduled", and records carry correlation IDs
// (notification_id, delivery_id, outbox_event_id, retry_cycle, attempt,
// task_id) so a single ID can be followed through the whole flow:
//
//	grep 'delivery_id=8f3c…' app.log
//	jq 'select(.notification_id=="…")' notification.log
//
// Configuration (environment):
//
//	NOTIFICATION_LOG_LEVEL   debug | info (default) | warn | error
//	NOTIFICATION_LOG_FORMAT  text (default) | json   — stdout format
//	NOTIFICATION_LOG_FILE    optional path; JSON lines are appended there
//	                         in addition to stdout, for analysis
//
// Privacy: never log notification bodies or HTML (they may contain OTPs).
// Use MaskEmail / MaskToken / MaskRecipient for recipients.
package notiflog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Correlation keys used across the flow. Keep names stable: they are the
// fields you filter on when analysing logs.
const (
	KeyNotificationID = "notification_id"
	KeyDeliveryID     = "delivery_id"
	KeyOutboxEventID  = "outbox_event_id"
	KeyEventType      = "event_type"
	KeyUserID         = "user_id"
	KeyChannel        = "channel"
	KeyRecipient      = "recipient"
	KeyRetryCycle     = "retry_cycle"
	KeyAttempt        = "attempt"
	KeyTaskID         = "task_id"
	KeyQueue          = "queue"
	KeyStatus         = "status"
	KeyError          = "error"
	KeyDurationMS     = "duration_ms"
	KeyNextRetryAt    = "next_retry_at"
	KeyRetryIn        = "retry_in"
	KeyAdminID        = "admin_id"
)

var current atomic.Pointer[slog.Logger]

func init() {
	// Default: colorized output when stdout is a terminal, plain text otherwise.
	if IsTerminal(os.Stdout) {
		current.Store(slog.New(NewColorHandler(os.Stdout, nil)))
	} else {
		current.Store(slog.New(slog.NewTextHandler(os.Stdout, nil)).With("flow", "notification"))
	}
}

// L returns the notification-flow logger.
func L() *slog.Logger {
	return current.Load()
}

// Init configures the logger from the environment. The returned function
// closes the log file, if one was opened.
func Init() (func() error, error) {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NOTIFICATION_LOG_LEVEL"))) {
	case "", "info":
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid NOTIFICATION_LOG_LEVEL %q", os.Getenv("NOTIFICATION_LOG_LEVEL"))
	}

	opts := &slog.HandlerOptions{Level: level}

	var stdout slog.Handler
	format := strings.ToLower(strings.TrimSpace(os.Getenv("NOTIFICATION_LOG_FORMAT")))
	switch format {
	case "", "auto":
		// Auto-detect: color if stdout is a terminal, plain text otherwise.
		if IsTerminal(os.Stdout) {
			stdout = NewColorHandler(os.Stdout, opts)
		} else {
			stdout = slog.NewTextHandler(os.Stdout, opts)
		}
	case "color":
		stdout = NewColorHandler(os.Stdout, opts)
	case "text":
		stdout = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		stdout = slog.NewJSONHandler(os.Stdout, opts)
	default:
		return nil, fmt.Errorf("invalid NOTIFICATION_LOG_FORMAT %q (use auto|color|text|json)", os.Getenv("NOTIFICATION_LOG_FORMAT"))
	}

	handlers := []slog.Handler{stdout}
	closer := func() error { return nil }

	if path := strings.TrimSpace(os.Getenv("NOTIFICATION_LOG_FILE")); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
		if err != nil {
			return nil, fmt.Errorf("open NOTIFICATION_LOG_FILE: %w", err)
		}

		handlers = append(handlers, slog.NewJSONHandler(file, opts))
		closer = file.Close
	}

	current.Store(slog.New(fanout(handlers)))

	L().Info("notiflog.configured",
		"log_level", level.String(),
		"log_format", format,
		"file", os.Getenv("NOTIFICATION_LOG_FILE"),
	)

	return closer, nil
}

// ---------------------------------------------------------------------
// Context-carried correlation fields
// ---------------------------------------------------------------------

type ctxKey struct{}

// With returns a context whose logger (From) includes the given
// key/value pairs, in addition to any added earlier.
func With(ctx context.Context, args ...any) context.Context {
	prev, _ := ctx.Value(ctxKey{}).([]any)
	merged := make([]any, 0, len(prev)+len(args))
	merged = append(merged, prev...)
	merged = append(merged, args...)
	return context.WithValue(ctx, ctxKey{}, merged)
}

// From returns the logger enriched with the context's correlation fields.
func From(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return L()
	}

	if args, ok := ctx.Value(ctxKey{}).([]any); ok && len(args) > 0 {
		return L().With(args...)
	}

	return L()
}

// ---------------------------------------------------------------------
// Attribute helpers
// ---------------------------------------------------------------------

// Since returns a duration_ms attribute measured from start.
func Since(start time.Time) slog.Attr {
	return slog.Int64(KeyDurationMS, time.Since(start).Milliseconds())
}

// Err returns an error attribute ("" when err is nil).
func Err(err error) slog.Attr {
	if err == nil {
		return slog.String(KeyError, "")
	}
	return slog.String(KeyError, err.Error())
}

// MaskEmail keeps the first character and the domain: "j***@gmail.com".
func MaskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}

// MaskToken keeps the first and last 4 characters of an FCM token.
func MaskToken(token string) string {
	if len(token) <= 12 {
		return "***"
	}
	return token[:4] + "…" + token[len(token)-4:]
}

// MaskRecipient masks a delivery recipient according to its channel.
func MaskRecipient(channel, recipient string) string {
	switch strings.ToUpper(channel) {
	case "EMAIL":
		return MaskEmail(recipient)
	default:
		return MaskToken(recipient)
	}
}

// ---------------------------------------------------------------------
// fanout handler: one record → several handlers (stdout + file)
// ---------------------------------------------------------------------

type fanoutHandler []slog.Handler

func fanout(handlers []slog.Handler) slog.Handler {
	if len(handlers) == 1 {
		return handlers[0]
	}
	return fanoutHandler(handlers)
}

func (f fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (f fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range f {
		if h.Enabled(ctx, r.Level) {
			errs = append(errs, h.Handle(ctx, r.Clone()))
		}
	}
	return errors.Join(errs...)
}

func (f fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make(fanoutHandler, len(f))
	for i, h := range f {
		out[i] = h.WithAttrs(attrs)
	}
	return out
}

func (f fanoutHandler) WithGroup(name string) slog.Handler {
	out := make(fanoutHandler, len(f))
	for i, h := range f {
		out[i] = h.WithGroup(name)
	}
	return out
}
