package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	. "github.com/go-jet/jet/v2/postgres"

	"github.com/Judeadeniji/zenv-sh/api/internal/audit"
	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// AuditHandler handles audit log query endpoints.
type AuditHandler struct {
	db     *sql.DB
	writer *audit.Writer
}

func NewAuditHandler(db *sql.DB, writer *audit.Writer) *AuditHandler {
	return &AuditHandler{db: db, writer: writer}
}

type AuditLogEntry struct {
	ID         string  `json:"id"`
	ProjectID  *string `json:"project_id,omitempty"`
	UserID     *string `json:"user_id,omitempty"`
	TokenID    *string `json:"token_id,omitempty"`
	ActorEmail *string `json:"actor_email,omitempty"`
	Action     string  `json:"action"`
	SecretHash *string `json:"secret_hash,omitempty"` // base64
	IP         *string `json:"ip,omitempty"`
	Result     string  `json:"result"`
	Metadata   *string `json:"metadata,omitempty"` // JSON string
	CreatedAt  string  `json:"created_at"`
}

type AuditLogListResponse struct {
	Entries []AuditLogEntry `json:"entries"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
}

// List returns paginated audit log entries filtered by query params.
//
//	@Summary		List audit logs
//	@Description	Query audit log entries with filters. Requires project_id.
//	@Tags			audit
//	@Produce		json
//	@Param			project_id	query		string	true	"Project ID"
//	@Param			start_date	query		string	false	"Start date (RFC3339)"
//	@Param			end_date	query		string	false	"End date (RFC3339)"
//	@Param			action		query		string	false	"Filter by action (e.g. secret.read)"
//	@Param			user_id		query		string	false	"Filter by user ID"
//	@Param			result		query		string	false	"Filter by result (success, denied, error)"
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			per_page	query		int		false	"Items per page (default 50, max 100)"
//	@Success		200			{object}	AuditLogListResponse
//	@Failure		400			{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/audit-logs [get]
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "project_id is required"})
		return
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project_id"})
		return
	}

	// Parse pagination.
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	// Build WHERE conditions.
	conditions := []BoolExpression{
		table.AuditLogs.ProjectID.EQ(UUID(projectID)),
	}

	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			conditions = append(conditions, table.AuditLogs.CreatedAt.GT_EQ(TimestampzT(t)))
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			conditions = append(conditions, table.AuditLogs.CreatedAt.LT_EQ(TimestampzT(t)))
		}
	}
	if action := r.URL.Query().Get("action"); action != "" {
		conditions = append(conditions, table.AuditLogs.Action.EQ(String(action)))
	}
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			conditions = append(conditions, table.AuditLogs.UserID.EQ(UUID(uid)))
		}
	}
	if result := r.URL.Query().Get("result"); result != "" {
		conditions = append(conditions, table.AuditLogs.Result.EQ(String(result)))
	}

	where := conditions[0]
	for _, c := range conditions[1:] {
		where = where.AND(c)
	}

	// Count total.
	var countResult struct{ Count int }
	countStmt := SELECT(COUNT(table.AuditLogs.ID).AS("count")).
		FROM(table.AuditLogs).WHERE(where)

	if err := countStmt.Query(h.db, &countResult); err != nil {
		slog.Error("audit-list: count", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to query audit logs"})
		return
	}

	// Fetch page with actor email via LEFT JOIN.
	offset := (page - 1) * perPage

	type auditRow struct {
		model.AuditLogs
		ActorEmail *string `alias:"users.email"`
	}
	var rows []auditRow

	stmt := SELECT(
		table.AuditLogs.AllColumns,
		table.Users.Email,
	).FROM(
		table.AuditLogs.LEFT_JOIN(table.Users, CAST(table.AuditLogs.UserID).AS_TEXT().EQ(table.Users.ID)),
	).WHERE(where).
		ORDER_BY(table.AuditLogs.CreatedAt.DESC()).
		LIMIT(int64(perPage)).
		OFFSET(int64(offset))

	if err := stmt.Query(h.db, &rows); err != nil {
		slog.Error("audit-list: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to query audit logs"})
		return
	}

	entries := make([]AuditLogEntry, 0, len(rows))
	for _, r := range rows {
		e := auditModelToEntry(r.AuditLogs)
		e.ActorEmail = r.ActorEmail
		entries = append(entries, e)
	}

	writeJSON(w, http.StatusOK, AuditLogListResponse{
		Entries: entries,
		Total:   countResult.Count,
		Page:    page,
		PerPage: perPage,
	})
}

// Export streams audit log entries as CSV.
//
//	@Summary		Export audit logs as CSV
//	@Description	Export filtered audit log entries as a CSV file.
//	@Tags			audit
//	@Produce		text/csv
//	@Param			project_id	query	string	true	"Project ID"
//	@Param			start_date	query	string	false	"Start date (RFC3339)"
//	@Param			end_date	query	string	false	"End date (RFC3339)"
//	@Param			action		query	string	false	"Filter by action"
//	@Param			user_id		query	string	false	"Filter by user ID"
//	@Param			result		query	string	false	"Filter by result"
//	@Success		200
//	@Security		SessionAuth
//	@Router			/audit-logs/export [get]
func (h *AuditHandler) Export(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "project_id is required"})
		return
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project_id"})
		return
	}

	// Build WHERE conditions (same logic as List).
	conditions := []BoolExpression{
		table.AuditLogs.ProjectID.EQ(UUID(projectID)),
	}
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			conditions = append(conditions, table.AuditLogs.CreatedAt.GT_EQ(TimestampzT(t)))
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			conditions = append(conditions, table.AuditLogs.CreatedAt.LT_EQ(TimestampzT(t)))
		}
	}
	if action := r.URL.Query().Get("action"); action != "" {
		conditions = append(conditions, table.AuditLogs.Action.EQ(String(action)))
	}
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			conditions = append(conditions, table.AuditLogs.UserID.EQ(UUID(uid)))
		}
	}
	if result := r.URL.Query().Get("result"); result != "" {
		conditions = append(conditions, table.AuditLogs.Result.EQ(String(result)))
	}

	where := conditions[0]
	for _, c := range conditions[1:] {
		where = where.AND(c)
	}

	var logs []model.AuditLogs
	stmt := SELECT(table.AuditLogs.AllColumns).
		FROM(table.AuditLogs).WHERE(where).
		ORDER_BY(table.AuditLogs.CreatedAt.DESC()).
		LIMIT(10000) // Safety cap

	if err := stmt.Query(h.db, &logs); err != nil {
		slog.Error("audit-export: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to query audit logs"})
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=audit_logs.csv")

	csvWriter := csv.NewWriter(w)
	csvWriter.Write([]string{"id", "project_id", "user_id", "token_id", "action", "secret_hash", "ip", "result", "created_at"})

	for _, l := range logs {
		csvWriter.Write([]string{
			uuidPtrStr(l.ID),
			uuidPtrStr(l.ProjectID),
			uuidPtrStr(l.UserID),
			uuidPtrStr(l.TokenID),
			l.Action,
			byteaToBase64(l.SecretHash),
			ptrStr(l.IP),
			l.Result,
			l.CreatedAt.Format(time.RFC3339),
		})
	}

	csvWriter.Flush()
}

func auditModelToEntry(l model.AuditLogs) AuditLogEntry {
	e := AuditLogEntry{
		ID:        uuidPtrStr(l.ID),
		Action:    l.Action,
		Result:    l.Result,
		CreatedAt: l.CreatedAt.Format(time.RFC3339),
	}
	if l.ProjectID != nil {
		s := l.ProjectID.String()
		e.ProjectID = &s
	}
	if l.UserID != nil {
		s := l.UserID.String()
		e.UserID = &s
	}
	if l.TokenID != nil {
		s := l.TokenID.String()
		e.TokenID = &s
	}
	if l.SecretHash != nil {
		s := base64.StdEncoding.EncodeToString(*l.SecretHash)
		e.SecretHash = &s
	}
	if l.IP != nil {
		e.IP = l.IP
	}
	if l.Metadata != nil {
		e.Metadata = l.Metadata
	}
	return e
}

func uuidPtrStr(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func byteaToBase64(b *[]byte) string {
	if b == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(*b)
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Drain flushes all queued audit events to Postgres immediately.
// Admin-only — gated by ADMIN_EMAILS env var.
//
//	@Summary		Drain audit queue
//	@Description	Force-flush all queued audit events from Redis to Postgres.
//	@Tags			audit
//	@Produce		json
//	@Success		200	{object}	map[string]int
//	@Failure		403	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/audit-logs/drain [post]
func (h *AuditHandler) Drain(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	if !isAdmin(sess.Email) {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "admin access required"})
		return
	}

	count, err := h.writer.Drain(r.Context())
	if err != nil {
		slog.Error("audit-drain: error", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "drain failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"flushed": count})
}

func isAdmin(email string) bool {
	admins := os.Getenv("ADMIN_EMAILS")
	if admins == "" {
		return false
	}
	for _, e := range strings.Split(admins, ",") {
		if strings.TrimSpace(e) == email {
			return true
		}
	}
	return false
}
