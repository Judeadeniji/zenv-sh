package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// TokensHandler handles service token CRUD.
// Tokens are hashed before storage — the plaintext is shown exactly once at creation.
type TokensHandler struct {
	db *sql.DB
}

func NewTokensHandler(db *sql.DB) *TokensHandler {
	return &TokensHandler{db: db}
}

// --- Create ---

type CreateTokenRequest struct {
	ProjectID   string  `json:"project_id"`
	Name        string  `json:"name"`
	Environment string  `json:"environment"`  // development | staging | production
	Permission  string  `json:"permission"`   // read | read_write
	ExpiresAt   *string `json:"expires_at"`   // optional RFC3339
}

type CreateTokenResponse struct {
	ID          string  `json:"id"`
	Token       string  `json:"token"`        // shown exactly once — never stored
	Name        string  `json:"name"`
	ProjectID   string  `json:"project_id"`
	Environment string  `json:"environment"`
	Permission  string  `json:"permission"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func (h *TokensHandler) Create(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	var req CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ProjectID == "" || req.Name == "" || req.Environment == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project_id, name, and environment are required"})
		return
	}

	if req.Permission == "" {
		req.Permission = "read"
	}
	if req.Permission != "read" && req.Permission != "read_write" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "permission must be 'read' or 'read_write'"})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project_id"})
		return
	}

	userID, err := uuid.Parse(sess.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid session"})
		return
	}

	// Generate token: svc_{env}_{random}
	tokenPlaintext, err := generateServiceToken(req.Environment)
	if err != nil {
		slog.Error("tokens.create: generate", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	// Hash before storage — server never stores plaintext.
	tokenHash := hashToken(tokenPlaintext)

	// Parse optional expiry.
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid expires_at — must be RFC3339"})
			return
		}
		expiresAt = &t
	}

	id := uuid.New()
	now := time.Now().UTC()

	insertStmt := table.ServiceTokens.INSERT(
		table.ServiceTokens.ID,
		table.ServiceTokens.ProjectID,
		table.ServiceTokens.Name,
		table.ServiceTokens.TokenHash,
		table.ServiceTokens.Environment,
		table.ServiceTokens.Permission,
		table.ServiceTokens.CreatedBy,
		table.ServiceTokens.CreatedAt,
	).VALUES(
		id, projectID, req.Name, tokenHash, req.Environment, req.Permission, userID, now,
	)

	// Add expiry if provided.
	if expiresAt != nil {
		insertStmt = table.ServiceTokens.INSERT(
			table.ServiceTokens.ID,
			table.ServiceTokens.ProjectID,
			table.ServiceTokens.Name,
			table.ServiceTokens.TokenHash,
			table.ServiceTokens.Environment,
			table.ServiceTokens.Permission,
			table.ServiceTokens.CreatedBy,
			table.ServiceTokens.ExpiresAt,
			table.ServiceTokens.CreatedAt,
		).VALUES(
			id, projectID, req.Name, tokenHash, req.Environment, req.Permission, userID, TimestampzT(*expiresAt), now,
		)
	}

	if _, err := insertStmt.Exec(h.db); err != nil {
		slog.Error("tokens.create: insert", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create token"})
		return
	}

	resp := CreateTokenResponse{
		ID:          id.String(),
		Token:       tokenPlaintext, // shown exactly once
		Name:        req.Name,
		ProjectID:   projectID.String(),
		Environment: req.Environment,
		Permission:  req.Permission,
		CreatedAt:   now.Format(time.RFC3339),
	}
	if req.ExpiresAt != nil {
		resp.ExpiresAt = req.ExpiresAt
	}

	writeJSON(w, http.StatusCreated, resp)
}

// --- Revoke ---

func (h *TokensHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	tokenID, err := uuid.Parse(chi.URLParam(r, "tokenID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid token ID"})
		return
	}

	now := time.Now().UTC()

	updateStmt := table.ServiceTokens.UPDATE(
		table.ServiceTokens.RevokedAt,
	).SET(
		TimestampzT(now),
	).WHERE(
		table.ServiceTokens.ID.EQ(UUID(tokenID)).
			AND(table.ServiceTokens.RevokedAt.IS_NULL()),
	)

	result, err := updateStmt.Exec(h.db)
	if err != nil {
		slog.Error("tokens.revoke: exec", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to revoke token"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "token not found or already revoked"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// --- List ---

type TokenListItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ProjectID   string  `json:"project_id"`
	Environment string  `json:"environment"`
	Permission  string  `json:"permission"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	RevokedAt   *string `json:"revoked_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type ListTokensResponse struct {
	Tokens []TokenListItem `json:"tokens"`
}

func (h *TokensHandler) List(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project_id query param is required"})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project_id"})
		return
	}

	var tokens []model.ServiceTokens
	stmt := SELECT(
		table.ServiceTokens.ID,
		table.ServiceTokens.Name,
		table.ServiceTokens.ProjectID,
		table.ServiceTokens.Environment,
		table.ServiceTokens.Permission,
		table.ServiceTokens.ExpiresAt,
		table.ServiceTokens.RevokedAt,
		table.ServiceTokens.CreatedAt,
	).FROM(table.ServiceTokens).WHERE(
		table.ServiceTokens.ProjectID.EQ(UUID(projectID)),
	).ORDER_BY(table.ServiceTokens.CreatedAt.DESC())

	if err := stmt.Query(h.db, &tokens); err != nil && err != sql.ErrNoRows {
		slog.Error("tokens.list: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list tokens"})
		return
	}

	result := make([]TokenListItem, 0, len(tokens))
	for _, t := range tokens {
		item := TokenListItem{
			ID:          t.ID.String(),
			Name:        t.Name,
			ProjectID:   t.ProjectID.String(),
			Environment: t.Environment,
			Permission:  t.Permission,
			CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		}
		if t.ExpiresAt != nil {
			s := t.ExpiresAt.Format(time.RFC3339)
			item.ExpiresAt = &s
		}
		if t.RevokedAt != nil {
			s := t.RevokedAt.Format(time.RFC3339)
			item.RevokedAt = &s
		}
		result = append(result, item)
	}

	writeJSON(w, http.StatusOK, ListTokensResponse{Tokens: result})
}

// --- Token generation and hashing ---

// generateServiceToken creates a token like "svc_dev_a3f9b2c8e1d4..."
func generateServiceToken(env string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token: rand: %w", err)
	}
	return fmt.Sprintf("svc_%s_%s", env, hex.EncodeToString(b)), nil
}

// hashToken returns SHA-256 of the plaintext token for storage.
func hashToken(plaintext string) []byte {
	h := sha256.Sum256([]byte(plaintext))
	return h[:]
}
