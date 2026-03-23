package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Color codes
const (
	reset   = "\033[0m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	gray    = "\033[90m"
)

// PrettyHandler is a compact, colorized slog handler for development.
type PrettyHandler struct {
	w     io.Writer
	mu    sync.Mutex
	level slog.Level
	attrs []slog.Attr
	group string
}

func NewPrettyHandler(w io.Writer, level slog.Level) *PrettyHandler {
	return &PrettyHandler{w: w, level: level}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder

	// Time — HH:MM:SS only
	b.WriteString(dim)
	b.WriteString(r.Time.Format("15:04:05"))
	b.WriteString(reset)
	b.WriteByte(' ')

	// Level — colored, 3 chars
	switch {
	case r.Level >= slog.LevelError:
		b.WriteString(red)
		b.WriteString("ERR")
	case r.Level >= slog.LevelWarn:
		b.WriteString(yellow)
		b.WriteString("WRN")
	case r.Level >= slog.LevelInfo:
		b.WriteString(green)
		b.WriteString("INF")
	default:
		b.WriteString(gray)
		b.WriteString("DBG")
	}
	b.WriteString(reset)
	b.WriteByte(' ')

	// Message
	msg := r.Message
	isRequest := msg == "request"

	// For request logs, format as: METHOD PATH STATUS DURATION
	if isRequest {
		var method, path, duration string
		var status int
		r.Attrs(func(a slog.Attr) bool {
			switch a.Key {
			case "method":
				method = a.Value.String()
			case "path":
				path = a.Value.String()
			case "status":
				status = int(a.Value.Int64())
			case "duration":
				duration = a.Value.String()
			}
			return true
		})

		// Method colored
		b.WriteString(methodColor(method))
		b.WriteString(method)
		b.WriteString(reset)
		b.WriteByte(' ')

		// Path
		b.WriteString(path)
		b.WriteByte(' ')

		// Status colored
		b.WriteString(statusColor(status))
		fmt.Fprintf(&b, "%d", status)
		b.WriteString(reset)
		b.WriteByte(' ')

		// Duration dimmed
		b.WriteString(dim)
		b.WriteString(formatDuration(duration))
		b.WriteString(reset)
	} else {
		// Regular message
		if r.Level >= slog.LevelError {
			b.WriteString(red)
		}
		b.WriteString(msg)
		if r.Level >= slog.LevelError {
			b.WriteString(reset)
		}

		// Attrs as key=value
		for _, a := range h.attrs {
			writeAttr(&b, a)
		}
		r.Attrs(func(a slog.Attr) bool {
			writeAttr(&b, a)
			return true
		})
	}

	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyHandler{
		w:     h.w,
		level: h.level,
		attrs: append(h.attrs, attrs...),
		group: h.group,
	}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{
		w:     h.w,
		level: h.level,
		attrs: h.attrs,
		group: name,
	}
}

func writeAttr(b *strings.Builder, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	b.WriteByte(' ')
	b.WriteString(cyan)
	b.WriteString(a.Key)
	b.WriteString(reset)
	b.WriteByte('=')
	v := a.Value.String()
	if strings.ContainsAny(v, " \t\n") {
		fmt.Fprintf(b, "%q", v)
	} else {
		b.WriteString(v)
	}
}

func methodColor(method string) string {
	switch method {
	case "GET":
		return blue
	case "POST":
		return green
	case "PUT":
		return yellow
	case "DELETE":
		return red
	default:
		return gray
	}
}

func statusColor(status int) string {
	switch {
	case status >= 500:
		return red
	case status >= 400:
		return yellow
	case status >= 300:
		return cyan
	default:
		return green
	}
}

func formatDuration(d string) string {
	// Parse and re-format for compactness
	dur, err := time.ParseDuration(d)
	if err != nil {
		return d
	}
	switch {
	case dur < time.Microsecond:
		return fmt.Sprintf("%dns", dur.Nanoseconds())
	case dur < time.Millisecond:
		return fmt.Sprintf("%dµs", dur.Microseconds())
	case dur < time.Second:
		return fmt.Sprintf("%dms", dur.Milliseconds())
	default:
		return fmt.Sprintf("%.1fs", dur.Seconds())
	}
}
