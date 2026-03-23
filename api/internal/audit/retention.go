package audit

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// RetentionPolicy configures audit log partition lifecycle.
type RetentionPolicy struct {
	RetainMonths int // Drop partitions older than this (default: 12)
	CreateAhead  int // Create partitions this many months ahead (default: 3)
}

// StartRetention runs a daily background goroutine that creates future
// partitions and drops expired ones. Cancel the context to stop.
func (w *Writer) StartRetention(ctx context.Context, policy RetentionPolicy) {
	if policy.RetainMonths <= 0 {
		policy.RetainMonths = 12
	}
	if policy.CreateAhead <= 0 {
		policy.CreateAhead = 3
	}

	// Run immediately on startup, then daily.
	go func() {
		w.runRetention(ctx, policy)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.runRetention(ctx, policy)
			}
		}
	}()

	slog.Debug("audit: retention worker started", "retain_months", policy.RetainMonths, "create_ahead", policy.CreateAhead)
}

func (w *Writer) runRetention(ctx context.Context, policy RetentionPolicy) {
	now := time.Now().UTC()

	// Create future partitions.
	for i := 0; i <= policy.CreateAhead; i++ {
		t := now.AddDate(0, i, 0)
		if err := w.createPartition(ctx, t); err != nil {
			slog.Error("audit: create partition", "month", t.Format("2006_01"), "error", err)
		}
	}

	// Drop expired partitions.
	cutoff := now.AddDate(0, -policy.RetainMonths, 0)
	// Scan back a generous window (36 months) to catch any old partitions.
	for i := policy.RetainMonths; i < policy.RetainMonths+36; i++ {
		t := now.AddDate(0, -i, 0)
		if t.After(cutoff) {
			continue
		}
		if err := w.dropPartition(ctx, t); err != nil {
			slog.Debug("audit: drop partition", "month", t.Format("2006_01"), "error", err)
		}
	}
}

func (w *Writer) createPartition(ctx context.Context, t time.Time) error {
	name := partitionName(t)
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	query := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_logs FOR VALUES FROM ('%s') TO ('%s')`,
		name,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
	)

	_, err := w.db.ExecContext(ctx, query)
	if err == nil {
		slog.Debug("audit: partition ensured", "name", name)
	}
	return err
}

func (w *Writer) dropPartition(ctx context.Context, t time.Time) error {
	name := partitionName(t)
	query := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, name)

	_, err := w.db.ExecContext(ctx, query)
	if err == nil {
		slog.Info("audit: partition dropped", "name", name)
	}
	return err
}

func partitionName(t time.Time) string {
	return fmt.Sprintf("audit_logs_%04d_%02d", t.Year(), t.Month())
}
