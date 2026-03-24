package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
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
	NameHash    string `json:"name_hash"`  // base64 HMAC-SHA256 of secret name
	Ciphertext  string `json:"ciphertext"` // base64 AES-256-GCM encrypted item JSON
	Nonce       string `json:"nonce"`      // base64 96-bit nonce
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

// @Summary		Create secret
// @Description	Store an encrypted vault item. Server stores opaque ciphertext only.
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			body	body		CreateSecretRequest	true	"Encrypted secret"
// @Success		201		{object}	SecretResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/sdk/secrets [post]
// @Router			/secrets [post]
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

// @Summary		Get secret
// @Description	Retrieve a single encrypted secret by name hash.
// @Tags			secrets
// @Produce		json
// @Param			nameHash	path		string	true	"HMAC-SHA256 name hash (base64)"
// @Param			project_id	query		string	true	"Project ID"
// @Param			environment	query		string	true	"Environment (development/staging/production)"
// @Success		200			{object}	SecretResponse
// @Failure		404			{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash} [get]
// @Router			/secrets/{nameHash} [get]
func (h *SecretsHandler) Get(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
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
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "secret not found"})
			return
		}
		slog.Error("secrets.get: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch secret"})
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

// @Summary		Bulk fetch secrets
// @Description	Fetch multiple secrets by name hashes. Used by SDK for schema manifest loading.
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			body	body		BulkFetchRequest	true	"Name hashes to fetch"
// @Success		200		{array}		SecretResponse
// @Security		BearerAuth
// @Router			/sdk/secrets/bulk [post]
// @Router			/secrets/bulk [post]
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
	Nonce      string `json:"nonce"`      // base64
}

// @Summary		Update secret
// @Description	Update ciphertext and nonce. Version auto-incremented.
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			nameHash	path		string				true	"HMAC-SHA256 name hash"
// @Param			project_id	query		string				true	"Project ID"
// @Param			environment	query		string				true	"Environment"
// @Param			body		body		UpdateSecretRequest	true	"New ciphertext"
// @Success		200			{object}	SecretResponse
// @Failure		404			{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash} [put]
// @Router			/secrets/{nameHash} [put]
func (h *SecretsHandler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
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

	// Fetch current version to archive before overwriting.
	var current model.VaultItems
	fetchCurrent := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(String(env))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)
	err = fetchCurrent.Query(h.db, &current)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "secret not found"})
		return
	}

	// Archive the current version before overwriting.
	archiveStmt := table.VaultItemVersions.INSERT(
		table.VaultItemVersions.ItemID,
		table.VaultItemVersions.Version,
		table.VaultItemVersions.Ciphertext,
		table.VaultItemVersions.Nonce,
		table.VaultItemVersions.CreatedAt,
	).VALUES(current.ID, current.Version, current.Ciphertext, current.Nonce, now)

	if _, err := archiveStmt.Exec(h.db); err != nil {
		slog.Error("secrets.update: archive version", "error", err)
		// Non-fatal — continue with update even if archiving fails.
	}

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
		table.VaultItems.ID.EQ(UUID(current.ID)),
	)

	if _, err := updateStmt.Exec(h.db); err != nil {
		slog.Error("secrets.update: exec", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update secret"})
		return
	}

	// Fetch updated row to return.
	var item model.VaultItems
	fetchStmt := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(table.VaultItems.ID.EQ(UUID(current.ID)))

	if err := fetchStmt.Query(h.db, &item); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		return
	}

	writeJSON(w, http.StatusOK, toSecretResponse(item))
}

// --- Delete ---

// @Summary		Delete secret
// @Description	Remove a secret from the vault.
// @Tags			secrets
// @Param			nameHash	path	string	true	"HMAC-SHA256 name hash"
// @Param			project_id	query	string	true	"Project ID"
// @Param			environment	query	string	true	"Environment"
// @Success			200
// @Failure			404	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash} [delete]
// @Router			/secrets/{nameHash} [delete]
func (h *SecretsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	slog.Info("secrets.delete: raw path param", "nameHash", nameHashB64)
	nameHash, err := decodeNameHash(nameHashB64)
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

// @Summary		List secrets
// @Description	List secret metadata (name hash, version, updated_at). Never returns ciphertext.
// @Tags			secrets
// @Produce		json
// @Param			project_id	query		string	true	"Project ID"
// @Param			environment	query		string	true	"Environment"
// @Success		200			{object}	ListSecretsResponse
// @Security		BearerAuth
// @Router			/sdk/secrets [get]
// @Router			/secrets [get]
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

	if err := stmt.Query(h.db, &items); err != nil && !errors.Is(err, qrm.ErrNoRows) {
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

// --- Versions ---

type VersionItem struct {
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
}

type VersionsResponse struct {
	Current  int           `json:"current_version"`
	Versions []VersionItem `json:"versions"`
}

// @Summary		List secret versions
// @Description	Show version history for a secret. Returns version numbers and timestamps.
// @Tags			secrets
// @Produce		json
// @Param			nameHash	path		string	true	"HMAC-SHA256 name hash"
// @Param			project_id	query		string	true	"Project ID"
// @Param			environment	query		string	true	"Environment"
// @Success		200			{object}	VersionsResponse
// @Failure		404			{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash}/versions [get]
// @Router			/secrets/{nameHash}/versions [get]
func (h *SecretsHandler) Versions(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid name_hash in URL"})
		return
	}

	// Get current item.
	var current model.VaultItems
	currentStmt := SELECT(table.VaultItems.ID, table.VaultItems.Version).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(String(env))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)

	if err := currentStmt.Query(h.db, &current); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "secret not found"})
		return
	}

	// Get archived versions.
	var archived []model.VaultItemVersions
	archiveStmt := SELECT(
		table.VaultItemVersions.Version,
		table.VaultItemVersions.CreatedAt,
	).FROM(table.VaultItemVersions).
		WHERE(table.VaultItemVersions.ItemID.EQ(UUID(current.ID))).
		ORDER_BY(table.VaultItemVersions.Version.DESC())

	_ = archiveStmt.Query(h.db, &archived)

	versions := make([]VersionItem, 0, len(archived))
	for _, v := range archived {
		versions = append(versions, VersionItem{
			Version:   int(v.Version),
			CreatedAt: v.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, VersionsResponse{
		Current:  int(current.Version),
		Versions: versions,
	})
}

// --- Rollback ---

type RollbackRequest struct {
	Version int `json:"version"`
}

// @Summary		Rollback secret
// @Description	Revert a secret to a previous version. The current version is archived first.
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			nameHash	path		string			true	"HMAC-SHA256 name hash"
// @Param			project_id	query		string			true	"Project ID"
// @Param			environment	query		string			true	"Environment"
// @Param			body		body		RollbackRequest	true	"Target version"
// @Success		200			{object}	SecretResponse
// @Failure		404			{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash}/rollback [post]
// @Router			/secrets/{nameHash}/rollback [post]
func (h *SecretsHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid name_hash in URL"})
		return
	}

	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Get current item.
	var current model.VaultItems
	currentStmt := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(String(env))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)

	if err := currentStmt.Query(h.db, &current); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "secret not found"})
		return
	}

	// Find the target version in archive.
	var target model.VaultItemVersions
	targetStmt := SELECT(table.VaultItemVersions.AllColumns).
		FROM(table.VaultItemVersions).
		WHERE(
			table.VaultItemVersions.ItemID.EQ(UUID(current.ID)).
				AND(table.VaultItemVersions.Version.EQ(Int(int64(req.Version)))),
		)

	if err := targetStmt.Query(h.db, &target); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error: fmt.Sprintf("version %d not found", req.Version),
		})
		return
	}

	now := time.Now().UTC()

	// Archive current version before rollback.
	archiveStmt := table.VaultItemVersions.INSERT(
		table.VaultItemVersions.ItemID,
		table.VaultItemVersions.Version,
		table.VaultItemVersions.Ciphertext,
		table.VaultItemVersions.Nonce,
		table.VaultItemVersions.CreatedAt,
	).VALUES(current.ID, current.Version, current.Ciphertext, current.Nonce, now)

	if _, err := archiveStmt.Exec(h.db); err != nil {
		slog.Error("secrets.rollback: archive current", "error", err)
	}

	// Overwrite with target version's ciphertext, bump version number.
	updateStmt := table.VaultItems.UPDATE(
		table.VaultItems.Ciphertext,
		table.VaultItems.Nonce,
		table.VaultItems.Version,
		table.VaultItems.UpdatedAt,
	).SET(
		Bytea(target.Ciphertext),
		Bytea(target.Nonce),
		table.VaultItems.Version.ADD(Int(1)),
		TimestampzT(now),
	).WHERE(table.VaultItems.ID.EQ(UUID(current.ID)))

	if _, err := updateStmt.Exec(h.db); err != nil {
		slog.Error("secrets.rollback: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "rollback failed"})
		return
	}

	// Return updated item.
	var item model.VaultItems
	fetchStmt := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(table.VaultItems.ID.EQ(UUID(current.ID)))

	if err := fetchStmt.Query(h.db, &item); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "rolled back"})
		return
	}

	writeJSON(w, http.StatusOK, toSecretResponse(item))
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

// decodeNameHash accepts both URL-safe and standard base64, with or without padding.
// It also URL-unescapes the input first, since HTTP clients may percent-encode
// characters like = (%3D) in path parameters.
func decodeNameHash(s string) ([]byte, error) {
	// URL-unescape first (handles %3D, %2B, %2F etc.)
	if decoded, err := url.PathUnescape(s); err == nil {
		s = decoded
	}

	// Try URL-safe base64 with padding
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// Try standard base64 with padding
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// Try URL-safe base64 without padding
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// Try standard base64 without padding
	return base64.RawStdEncoding.DecodeString(s)
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
