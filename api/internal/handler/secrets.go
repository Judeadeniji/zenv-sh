package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// SecretsHandler handles encrypted vault item CRUD.
// The server never sees plaintext — it stores and returns opaque ciphertext.
type SecretsHandler struct {
	db *sql.DB
}

func NewSecretsHandler(db *sql.DB) *SecretsHandler {
	return &SecretsHandler{db: db}
}

// --- Create ---

type CreateSecretRequest struct {
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
	NameHash    string `json:"name_hash"`   // base64 HMAC-SHA256 of secret name
	Ciphertext  string `json:"ciphertext"`  // base64 AES-256-GCM encrypted item JSON
	Nonce       string `json:"nonce"`        // base64 96-bit nonce
}

type SecretResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
	NameHash    string `json:"name_hash"`
	Ciphertext  string `json:"ciphertext"`
	Nonce       string `json:"nonce"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (h *SecretsHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Auth is enforced by middleware (session or token) before this handler runs.
	var req CreateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ProjectID == "" || req.Environment == "" || req.NameHash == "" || req.Ciphertext == "" || req.Nonce == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "all fields are required"})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project_id"})
		return
	}

	nameHash, err := base64.StdEncoding.DecodeString(req.NameHash)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in name_hash"})
		return
	}
	ciphertext, err := base64.StdEncoding.DecodeString(req.Ciphertext)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in ciphertext"})
		return
	}
	nonce, err := base64.StdEncoding.DecodeString(req.Nonce)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in nonce"})
		return
	}

	// Check if secret with same name_hash already exists in this project+environment.
	var existing model.VaultItems
	existsStmt := SELECT(table.VaultItems.ID).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(String(req.Environment))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)

	if err := existsStmt.Query(h.db, &existing); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "secret already exists — use PUT to update"})
		return
	}

	id := uuid.New()
	now := time.Now().UTC()

	insertStmt := table.VaultItems.INSERT(
		table.VaultItems.ID,
		table.VaultItems.ProjectID,
		table.VaultItems.Environment,
		table.VaultItems.NameHash,
		table.VaultItems.Ciphertext,
		table.VaultItems.Nonce,
		table.VaultItems.Version,
		table.VaultItems.CreatedAt,
		table.VaultItems.UpdatedAt,
	).VALUES(
		id, projectID, req.Environment, nameHash, ciphertext, nonce, 1, now, now,
	)

	if _, err := insertStmt.Exec(h.db); err != nil {
		slog.Error("secrets.create: insert", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store secret"})
		return
	}

	writeJSON(w, http.StatusCreated, SecretResponse{
		ID:          id.String(),
		ProjectID:   projectID.String(),
		Environment: req.Environment,
		NameHash:    req.NameHash,
		Ciphertext:  req.Ciphertext,
		Nonce:       req.Nonce,
		Version:     1,
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	})
}

// --- Get single secret by name_hash ---

func (h *SecretsHandler) Get(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := base64.URLEncoding.DecodeString(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid name_hash in URL"})
		return
	}

	var item model.VaultItems
	stmt := SELECT(
		table.VaultItems.AllColumns,
	).FROM(table.VaultItems).WHERE(
		table.VaultItems.ProjectID.EQ(UUID(projectID)).
			AND(table.VaultItems.Environment.EQ(String(env))).
			AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
	)

	if err := stmt.Query(h.db, &item); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "secret not found"})
		return
	}

	writeJSON(w, http.StatusOK, toSecretResponse(item))
}

// --- Bulk fetch by name hashes (schema manifest) ---

type BulkFetchRequest struct {
	ProjectID   string   `json:"project_id"`
	Environment string   `json:"environment"`
	NameHashes  []string `json:"name_hashes"` // base64-encoded HMAC hashes
}

type BulkFetchResponse struct {
	Secrets []SecretResponse `json:"secrets"`
}

func (h *SecretsHandler) BulkFetch(w http.ResponseWriter, r *http.Request) {
	var req BulkFetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project_id"})
		return
	}

	if len(req.NameHashes) == 0 {
		writeJSON(w, http.StatusOK, BulkFetchResponse{Secrets: []SecretResponse{}})
		return
	}

	// Decode all name hashes and build IN clause.
	hashExpressions := make([]Expression, 0, len(req.NameHashes))
	for _, h64 := range req.NameHashes {
		decoded, err := base64.StdEncoding.DecodeString(h64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in name_hashes"})
			return
		}
		hashExpressions = append(hashExpressions, Bytea(decoded))
	}

	var items []model.VaultItems
	stmt := SELECT(
		table.VaultItems.AllColumns,
	).FROM(table.VaultItems).WHERE(
		table.VaultItems.ProjectID.EQ(UUID(projectID)).
			AND(table.VaultItems.Environment.EQ(String(req.Environment))).
			AND(table.VaultItems.NameHash.IN(hashExpressions...)),
	)

	if err := stmt.Query(h.db, &items); err != nil {
		slog.Error("secrets.bulk_fetch: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch secrets"})
		return
	}

	secrets := make([]SecretResponse, 0, len(items))
	for _, item := range items {
		secrets = append(secrets, toSecretResponse(item))
	}

	writeJSON(w, http.StatusOK, BulkFetchResponse{Secrets: secrets})
}

// --- Update (new version, new nonce) ---

type UpdateSecretRequest struct {
	Ciphertext string `json:"ciphertext"` // base64
	Nonce      string `json:"nonce"`       // base64
}

func (h *SecretsHandler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := base64.URLEncoding.DecodeString(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid name_hash in URL"})
		return
	}

	var req UpdateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	ciphertext, err := base64.StdEncoding.DecodeString(req.Ciphertext)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in ciphertext"})
		return
	}
	nonce, err := base64.StdEncoding.DecodeString(req.Nonce)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in nonce"})
		return
	}

	now := time.Now().UTC()

	updateStmt := table.VaultItems.UPDATE(
		table.VaultItems.Ciphertext,
		table.VaultItems.Nonce,
		table.VaultItems.Version,
		table.VaultItems.UpdatedAt,
	).SET(
		Bytea(ciphertext),
		Bytea(nonce),
		table.VaultItems.Version.ADD(Int(1)),
		TimestampzT(now),
	).WHERE(
		table.VaultItems.ProjectID.EQ(UUID(projectID)).
			AND(table.VaultItems.Environment.EQ(String(env))).
			AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
	)

	result, err := updateStmt.Exec(h.db)
	if err != nil {
		slog.Error("secrets.update: exec", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update secret"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "secret not found"})
		return
	}

	// Fetch updated row to return.
	var item model.VaultItems
	fetchStmt := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(String(env))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)

	if err := fetchStmt.Query(h.db, &item); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		return
	}

	writeJSON(w, http.StatusOK, toSecretResponse(item))
}

// --- Delete ---

func (h *SecretsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := base64.URLEncoding.DecodeString(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid name_hash in URL"})
		return
	}

	deleteStmt := table.VaultItems.DELETE().WHERE(
		table.VaultItems.ProjectID.EQ(UUID(projectID)).
			AND(table.VaultItems.Environment.EQ(String(env))).
			AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
	)

	result, err := deleteStmt.Exec(h.db)
	if err != nil {
		slog.Error("secrets.delete: exec", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete secret"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "secret not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- List (names only — never values) ---

type ListSecretsResponse struct {
	Secrets []SecretListItem `json:"secrets"`
}

type SecretListItem struct {
	ID          string `json:"id"`
	NameHash    string `json:"name_hash"`
	Environment string `json:"environment"`
	Version     int    `json:"version"`
	UpdatedAt   string `json:"updated_at"`
}

func (h *SecretsHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var items []model.VaultItems
	stmt := SELECT(
		table.VaultItems.ID,
		table.VaultItems.NameHash,
		table.VaultItems.Environment,
		table.VaultItems.Version,
		table.VaultItems.UpdatedAt,
	).FROM(table.VaultItems).WHERE(
		table.VaultItems.ProjectID.EQ(UUID(projectID)).
			AND(table.VaultItems.Environment.EQ(String(env))),
	).ORDER_BY(table.VaultItems.UpdatedAt.DESC())

	if err := stmt.Query(h.db, &items); err != nil && err != sql.ErrNoRows {
		slog.Error("secrets.list: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list secrets"})
		return
	}

	result := make([]SecretListItem, 0, len(items))
	for _, item := range items {
		result = append(result, SecretListItem{
			ID:          item.ID.String(),
			NameHash:    base64.StdEncoding.EncodeToString(item.NameHash),
			Environment: item.Environment,
			Version:     int(item.Version),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, ListSecretsResponse{Secrets: result})
}

// --- Helpers ---

func parseProjectEnv(r *http.Request) (uuid.UUID, string, error) {
	pidStr := r.URL.Query().Get("project_id")
	env := r.URL.Query().Get("environment")

	if pidStr == "" || env == "" {
		return uuid.UUID{}, "", fmt.Errorf("project_id and environment query params are required")
	}

	pid, err := uuid.Parse(pidStr)
	if err != nil {
		return uuid.UUID{}, "", fmt.Errorf("invalid project_id")
	}

	return pid, env, nil
}

func toSecretResponse(item model.VaultItems) SecretResponse {
	return SecretResponse{
		ID:          item.ID.String(),
		ProjectID:   item.ProjectID.String(),
		Environment: item.Environment,
		NameHash:    base64.StdEncoding.EncodeToString(item.NameHash),
		Ciphertext:  base64.StdEncoding.EncodeToString(item.Ciphertext),
		Nonce:       base64.StdEncoding.EncodeToString(item.Nonce),
		Version:     int(item.Version),
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}
}
