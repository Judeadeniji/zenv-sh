// Package audit provides a non-blocking audit log writer.
//
// Architecture (from master plan Section 8.9):
//   API handler completes → LPUSH audit:queue (microseconds, fire-and-forget)
//   → API returns response immediately
//   → In-process worker: BRPOP audit:queue, batches up to 100 events or 5 seconds
//   → Single bulk INSERT into Postgres audit_logs
//
// At 1000 events/second, Postgres sees at most 12 bulk inserts per minute.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	queueKey      = "audit:queue"
	batchSize     = 100
	flushInterval = 5 * time.Second
)

// Event represents a single audit log entry.
type Event struct {
	ProjectID  *uuid.UUID      `json:"project_id,omitempty"`
	UserID     *uuid.UUID      `json:"user_id,omitempty"`
	TokenID    *uuid.UUID      `json:"token_id,omitempty"`
	Action     string          `json:"action"`      // e.g. "secret.read", "secret.write", "token.create"
	SecretHash []byte          `json:"secret_hash,omitempty"`
	IP         string          `json:"ip,omitempty"`
	UserAgent  string          `json:"user_agent,omitempty"`
	Result     string          `json:"result"`      // "success", "denied", "error"
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// Writer is the audit log writer. Call Emit() to queue events,
// Start() to begin the background flush worker.
type Writer struct {
	rdb    *redis.Client
	db     *sql.DB
	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a new audit log writer.
func New(db *sql.DB, rdb *redis.Client) *Writer {
	return &Writer{db: db, rdb: rdb, done: make(chan struct{})}
}

// Emit queues an audit event. Non-blocking — returns immediately.
// If Redis is down, the event is silently dropped (logged at debug level).
func (w *Writer) Emit(ctx context.Context, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Debug("audit: marshal event", "error", err)
		return
	}

	if err := w.rdb.LPush(ctx, queueKey, data).Err(); err != nil {
		slog.Debug("audit: lpush", "error", err)
	}
}

// EmitFromRequest is a convenience method that extracts IP and User-Agent from the request.
func (w *Writer) EmitFromRequest(r *http.Request, event Event) {
	event.IP = extractIP(r)
	event.UserAgent = r.UserAgent()
	w.Emit(r.Context(), event)
}

// Start begins the background flush worker. Call this once at startup.
func (w *Writer) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	go func() {
		w.worker(ctx)
		close(w.done)
	}()
	slog.Debug("audit: worker started")
}

// Stop signals the worker to stop and waits for it to drain remaining events.
func (w *Writer) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	<-w.done
	slog.Info("audit: worker stopped, queue drained")
}

// Drain immediately flushes all events currently in the Redis queue to Postgres.
// Safe to call concurrently with the background worker.
func (w *Writer) Drain(ctx context.Context) (int, error) {
	total := 0
	for {
		result, err := w.rdb.RPop(ctx, queueKey).Result()
		if err != nil {
			break // queue empty or error
		}
		var event Event
		if err := json.Unmarshal([]byte(result), &event); err != nil {
			slog.Debug("audit: drain unmarshal", "error", err)
			continue
		}
		w.flush([]Event{event})
		total++
	}
	return total, nil
}

func (w *Writer) worker(ctx context.Context) {
	batch := make([]Event, 0, batchSize)
	timer := time.NewTimer(flushInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// Flush remaining events before shutdown.
			if len(batch) > 0 {
				w.flush(batch)
			}
			slog.Debug("audit: worker stopped")
			return

		case <-timer.C:
			// Flush on timer even if batch isn't full.
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
			timer.Reset(flushInterval)

		default:
			// BRPOP with short timeout so we check ctx.Done() periodically.
			result, err := w.rdb.BRPop(ctx, 1*time.Second, queueKey).Result()
			if err != nil {
				// Timeout or context cancelled — loop back to check.
				continue
			}

			// result[0] is the key name, result[1] is the value.
			var event Event
			if err := json.Unmarshal([]byte(result[1]), &event); err != nil {
				slog.Debug("audit: unmarshal event", "error", err)
				continue
			}

			batch = append(batch, event)

			if len(batch) >= batchSize {
				w.flush(batch)
				batch = batch[:0]
				timer.Reset(flushInterval)
			}
		}
	}
}

func (w *Writer) flush(events []Event) {
	if len(events) == 0 {
		return
	}

	// Build bulk INSERT.
	// INSERT INTO audit_logs (project_id, user_id, token_id, action, secret_hash, ip, user_agent, result, metadata)
	// VALUES ($1, $2, ...), ($N, ...);
	var b strings.Builder
	b.WriteString("INSERT INTO audit_logs (project_id, user_id, token_id, action, secret_hash, ip, user_agent, result, metadata, created_at) VALUES ")

	args := make([]any, 0, len(events)*10)
	for i, e := range events {
		if i > 0 {
			b.WriteString(", ")
		}
		offset := i * 10
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5,
			offset+6, offset+7, offset+8, offset+9, offset+10)

		var metadata *json.RawMessage
		if len(e.Metadata) > 0 {
			metadata = &e.Metadata
		}

		args = append(args,
			uuidOrNil(e.ProjectID),
			uuidOrNil(e.UserID),
			uuidOrNil(e.TokenID),
			e.Action,
			e.SecretHash,
			nilIfEmpty(e.IP),
			nilIfEmpty(e.UserAgent),
			e.Result,
			metadata,
			time.Now().UTC(),
		)
	}

	_, err := w.db.Exec(b.String(), args...)
	if err != nil {
		slog.Error("audit: flush", "error", err, "count", len(events))
		return
	}

	slog.Debug("audit: flushed", "count", len(events))
}

func uuidOrNil(u *uuid.UUID) any {
	if u == nil {
		return nil
	}
	return *u
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func extractIP(r *http.Request) string {
	// Check X-Forwarded-For first (for reverse proxies).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
