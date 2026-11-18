package notiflog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[90m"

	// Bold variants
	boldRed    = "\033[1;31m"
	boldGreen  = "\033[1;32m"
	boldYellow = "\033[1;33m"
	boldBlue   = "\033[1;34m"
	boldCyan   = "\033[1;36m"
	boldWhite  = "\033[1;37m"

	// Dim
	dimWhite = "\033[2;37m"
)

// Step icons for the notification lifecycle.
var stepIcons = map[string]string{
	// ── Send (API entry) ──
	"send.started":                "📨",
	"send.validation_failed":      "❌",
	"send.channel_skipped":        "⏭️",
	"send.recipient_resolution_failed": "❌",
	"send.email_lookup_failed":    "❌",
	"send.device_lookup_failed":   "❌",
	"send.channel_resolved":       "✅",
	"send.no_recipients":          "⚠️",
	"send.store_failed":           "❌",
	"send.stored":                 "💾",
	"send.delivery_created":       "📋",

	// ── Outbox ──
	"outbox.worker_started":           "🔄",
	"outbox.worker_stopped":           "🛑",
	"outbox.events_claimed":           "📥",
	"outbox.event_processing_started": "⚙️",
	"outbox.heartbeat_start_failed":   "❌",
	"outbox.ownership_lost":           "🔒",
	"outbox.event_interrupted":        "⏸️",
	"outbox.mark_processed_failed":    "❌",
	"outbox.event_processed":          "✅",
	"outbox.event_retry_scheduled":    "🔁",
	"outbox.event_dead":               "💀",
	"outbox.mark_dead_failed":         "❌",
	"outbox.mark_failed_failed":       "❌",
	"outbox.handler_registered":       "📌",
	"outbox.no_handler_registered":    "❌",
	"outbox.claim_failed":             "❌",
	"outbox.event_failed":             "❌",

	// ── Notification Created Event ──
	"notification_created.publishing":        "📡",
	"notification_created.publish_incomplete": "⚠️",
	"notification_created.published":         "✅",

	// ── Queue ──
	"queue.task_enqueued":          "📤",
	"queue.duplicate_task_skipped": "♻️",
	"queue.enqueue_failed":         "❌",

	// ── Task (Asynq worker) ──
	"task.received":          "📩",
	"task.payload_invalid":   "❌",
	"task.acknowledged_stale": "♻️",
	"task.failed":            "❌",
	"task.completed":         "✅",

	// ── Delivery processing ──
	"delivery.processing_started":          "🚀",
	"delivery.task_invalid":                "❌",
	"delivery.load_failed":                 "❌",
	"delivery.loaded":                      "📄",
	"delivery.stale_task_ignored":          "♻️",
	"delivery.task_mismatch":               "❌",
	"delivery.claim_failed":                "❌",
	"delivery.claim_skipped":               "⏭️",
	"delivery.claimed":                     "🔐",
	"delivery.notification_load_failed":    "❌",
	"delivery.notification_missing":        "❌",
	"delivery.attempt_started":             "🎯",
	"delivery.attempts_exhausted_before_send": "💀",
	"delivery.attempt_create_failed":       "❌",
	"delivery.dead_letter_failed":          "❌",
	"delivery.dead_lettered":               "💀",
	"delivery.ownership_lost":              "🔒",
	"delivery.retry_schedule_failed":       "❌",
	"delivery.retry_scheduled":             "🔁",
	"delivery.failed_permanently":          "💥",
	"delivery.outcome_unknown":             "❓",
	"delivery.unsupported_decision":        "❌",
	"delivery.attempt_update_failed":       "❌",
	"delivery.status_update_failed":        "❌",
	"delivery.provider_id_save_failed":     "❌",
	"delivery.accepted":                    "✅",
	"delivery.heartbeat_failed":            "❌",
	"delivery.heartbeat_ownership_lost":    "🔒",
	"delivery.heartbeat_renewed":           "💓",

	// ── Provider ──
	"provider.send_started":    "📡",
	"provider.send_succeeded":  "✅",
	"provider.send_failed":     "❌",
	"provider.error_classified": "🏷️",

	// ── Device ──
	"device.deactivated_unregistered_token": "📵",
	"device.deactivate_failed":              "❌",

	// ── DLQ retry ──
	"dlq_retry.event_obsolete": "♻️",
	"dlq_retry.publish_failed": "❌",
	"dlq_retry.published":      "✅",
}

// ColorHandler is a slog.Handler that writes colorized, human-friendly
// notification lifecycle logs to a terminal.
//
// Log format:
//
//	23:11:15.226 INFO  ✅ delivery.accepted  delivery_id=7c11… channel=EMAIL duration_ms=1127
type ColorHandler struct {
	opts      slog.HandlerOptions
	w         io.Writer
	mu        sync.Mutex
	preformat []slog.Attr // attrs from WithAttrs
	groups    []string    // groups from WithGroup
}

// NewColorHandler creates a handler that writes colorized logs to w.
func NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ColorHandler{
		opts: *opts,
		w:    w,
	}
}

func (h *ColorHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *ColorHandler) Handle(_ context.Context, r slog.Record) error {
	var buf strings.Builder

	// ── Timestamp (gray, compact) ──
	buf.WriteString(colorGray)
	buf.WriteString(r.Time.Format("15:04:05.000"))
	buf.WriteString(colorReset)
	buf.WriteByte(' ')

	// ── Level (colored) ──
	levelColor, levelLabel := levelStyle(r.Level)
	buf.WriteString(levelColor)
	buf.WriteString(levelLabel)
	buf.WriteString(colorReset)
	buf.WriteByte(' ')

	// ── Icon ──
	if icon, ok := stepIcons[r.Message]; ok {
		buf.WriteString(icon)
		buf.WriteByte(' ')
	}

	// ── Message (bold white) ──
	buf.WriteString(boldWhite)
	buf.WriteString(r.Message)
	buf.WriteString(colorReset)

	// ── Preformatted attrs from With() ──
	for _, attr := range h.preformat {
		writeAttr(&buf, attr, r.Level)
	}

	// ── Record attrs ──
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&buf, a, r.Level)
		return true
	})

	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, buf.String())
	return err
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := h.clone()
	h2.preformat = append(h2.preformat, attrs...)
	return h2
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *ColorHandler) clone() *ColorHandler {
	return &ColorHandler{
		opts:      h.opts,
		w:         h.w,
		preformat: append([]slog.Attr(nil), h.preformat...),
		groups:    append([]string(nil), h.groups...),
	}
}

// ── Helpers ──

func levelStyle(level slog.Level) (color, label string) {
	switch {
	case level >= slog.LevelError:
		return boldRed, "ERROR"
	case level >= slog.LevelWarn:
		return boldYellow, "WARN "
	case level >= slog.LevelInfo:
		return boldGreen, "INFO "
	default:
		return colorGray, "DEBUG"
	}
}

// highlightKeys are rendered in color instead of gray.
var highlightKeys = map[string]string{
	KeyNotificationID: colorCyan,
	KeyDeliveryID:     colorCyan,
	KeyOutboxEventID:  colorCyan,
	KeyChannel:        colorBlue,
	KeyStatus:         colorYellow,
	KeyUserID:         colorCyan,
	KeyRecipient:      colorCyan,
	KeyAttempt:        colorYellow,
	KeyRetryCycle:     colorYellow,
	KeyDurationMS:     colorGreen,
	KeyError:          colorRed,
	KeyNextRetryAt:    colorYellow,
	KeyRetryIn:        colorYellow,
	KeyTaskID:         colorGray,
	KeyQueue:          colorGray,
	"priority":        colorBlue,
	"notification_type": colorBlue,
	"decision":        colorYellow,
	"reason":          colorYellow,
	"action":          colorYellow,
}

func writeAttr(buf *strings.Builder, attr slog.Attr, level slog.Level) {
	if attr.Equal(slog.Attr{}) {
		return
	}

	key := attr.Key
	val := attr.Value.Resolve()

	// Skip empty error attributes.
	if key == KeyError && val.String() == "" {
		return
	}

	buf.WriteByte(' ')

	// Key coloring.
	if c, ok := highlightKeys[key]; ok {
		buf.WriteString(dimWhite)
		buf.WriteString(key)
		buf.WriteString("=")
		buf.WriteString(colorReset)
		buf.WriteString(c)
		buf.WriteString(formatValue(val))
		buf.WriteString(colorReset)
	} else {
		buf.WriteString(dimWhite)
		buf.WriteString(key)
		buf.WriteString("=")
		buf.WriteString(formatValue(val))
		buf.WriteString(colorReset)
	}
}

func formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	case slog.KindDuration:
		return v.Duration().String()
	default:
		return fmt.Sprintf("%v", v.Any())
	}
}

// IsTerminal checks if w is a terminal that supports colors.
func IsTerminal(w *os.File) bool {
	info, err := w.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

