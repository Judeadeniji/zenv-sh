package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ProjectID   string            `json:"project_id"`
	Environment model.Environment `json:"environment"`
	NameHash    string            `json:"name_hash"`  // base64 HMAC-SHA256 of secret name
	Ciphertext  string            `json:"ciphertext"` // base64 AES-256-GCM encrypted item JSON
	Nonce       string            `json:"nonce"`      // base64 96-bit nonce
	// Metadata is optional plaintext hints (MIME, description, tags). Never put secret material here.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type SecretResponse struct {
	ID          string            `json:"id"`
	ProjectID   string            `json:"project_id"`
	Environment model.Environment `json:"environment"`
	NameHash    string            `json:"name_hash"`
	Ciphertext  string            `json:"ciphertext"`
	Nonce       string            `json:"nonce"`
	Metadata    json.RawMessage   `json:"metadata,omitempty"`
	Version     int               `json:"version"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// @Summary		Create secret
// @Description	Store an encrypted vault item. The server stores opaque ciphertext and nonce only; the secret name is never sent in plaintext (HMAC-SHA256 name_hash). Optional `metadata` is plaintext JSON (mime_type, description, tags, labels) for operators and tooling — never put secret values in metadata.
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			body	body		CreateSecretRequest	true	"Encrypted secret payload"
// @Success		201		{object}	SecretResponse
// @Failure		400		{object}	ErrorResponse	"Missing or invalid fields, or invalid base64 encoding"
// @Failure		409		{object}	ErrorResponse	"Secret with this name hash already exists in the given project+environment — use PUT to update"
// @Failure		500		{object}	ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets [post]
// @Router			/secrets [post]
func (h *SecretsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.ProjectID == "" || req.Environment == "" || req.NameHash == "" || req.Ciphertext == "" || req.Nonce == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "project_id, environment, name_hash, ciphertext, and nonce are all required"})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project_id"})
		return
	}

	nameHash, err := base64.StdEncoding.DecodeString(req.NameHash)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in name_hash"})
		return
	}
	ciphertext, err := base64.StdEncoding.DecodeString(req.Ciphertext)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in ciphertext"})
		return
	}
	nonce, err := base64.StdEncoding.DecodeString(req.Nonce)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in nonce"})
		return
	}

	// Check for duplicate name_hash in this project+environment.
	var existing model.VaultItems
	existsStmt := SELECT(table.VaultItems.ID).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(NewEnumValue(req.Environment.String()))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)
	if err := existsStmt.Query(h.db, &existing); err == nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "secret already exists — use PUT to update"})
		return
	}

	id := uuid.New()
	now := time.Now().UTC()

	metaJSON := "{}"
	if len(req.Metadata) > 0 {
		mb, err := normalizeSecretMetadata(req.Metadata)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		metaJSON = string(mb)
	}

	insertStmt := table.VaultItems.INSERT(
		table.VaultItems.ID,
		table.VaultItems.ProjectID,
		table.VaultItems.Environment,
		table.VaultItems.NameHash,
		table.VaultItems.Ciphertext,
		table.VaultItems.Nonce,
		table.VaultItems.Version,
		table.VaultItems.Metadata,
		table.VaultItems.CreatedAt,
		table.VaultItems.UpdatedAt,
	).VALUES(
		id, projectID, req.Environment, nameHash, ciphertext, nonce, 1, String(metaJSON), now, now,
	)

	if _, err := insertStmt.Exec(h.db); err != nil {
		slog.Error("secrets.create: insert", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to store secret"})
		return
	}

	var created model.VaultItems
	if err := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(table.VaultItems.ID.EQ(UUID(id))).
		Query(h.db, &created); err != nil {
		slog.Error("secrets.create: fetch", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to load secret"})
		return
	}

	writeJSON(w, http.StatusCreated, toSecretResponse(created))
}

// --- Get single secret by name_hash ---

// @Summary		Get secret
// @Description	Retrieve one encrypted secret by HMAC-SHA256 name hash (must match exactly). Returns ciphertext, nonce, version, timestamps, and optional plaintext metadata. Partial or plaintext name lookups are not supported.
// @Tags			secrets
// @Produce		json
// @Param			nameHash	path		string	true	"HMAC-SHA256 name hash (base64, URL-encoded)"
// @Param			project_id	query		string	true	"Project ID"
// @Param			environment	query		string	true	"Environment (development | staging | production)"
// @Success		200			{object}	SecretResponse
// @Failure		400			{object}	ErrorResponse	"Missing query params or invalid name hash encoding"
// @Failure		404			{object}	ErrorResponse	"Secret not found"
// @Failure		500			{object}	ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash} [get]
// @Router			/secrets/{nameHash} [get]
func (h *SecretsHandler) Get(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid name_hash in URL"})
		return
	}

	var item model.VaultItems
	stmt := SELECT(
		table.VaultItems.AllColumns,
	).FROM(table.VaultItems).WHERE(
		table.VaultItems.ProjectID.EQ(UUID(projectID)).
			AND(table.VaultItems.Environment.EQ(NewEnumValue(env))).
			AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
	)

	if err := stmt.Query(h.db, &item); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "secret not found"})
			return
		}
		slog.Error("secrets.get: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch secret"})
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
// @Description	Fetch multiple encrypted secrets in one request by providing a list of HMAC-SHA256 name hashes. Used by clients for manifest loading. Each item in the response includes optional plaintext `metadata` alongside ciphertext. Missing hashes are omitted from the result (no error).
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			body	body		BulkFetchRequest	true	"Project, environment, and list of name hashes to fetch"
// @Success		200		{object}	BulkFetchResponse
// @Failure		400		{object}	ErrorResponse	"Invalid project_id or malformed base64 in name_hashes"
// @Failure		500		{object}	ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets/bulk [post]
// @Router			/secrets/bulk [post]
func (h *SecretsHandler) BulkFetch(w http.ResponseWriter, r *http.Request) {
	var req BulkFetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project_id"})
		return
	}

	if len(req.NameHashes) == 0 {
		writeJSON(w, http.StatusOK, BulkFetchResponse{Secrets: []SecretResponse{}})
		return
	}

	hashExpressions := make([]Expression, 0, len(req.NameHashes))
	for _, h64 := range req.NameHashes {
		decoded, err := base64.StdEncoding.DecodeString(h64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in name_hashes"})
			return
		}
		hashExpressions = append(hashExpressions, Bytea(decoded))
	}

	var items []model.VaultItems
	stmt := SELECT(
		table.VaultItems.AllColumns,
	).FROM(table.VaultItems).WHERE(
		table.VaultItems.ProjectID.EQ(UUID(projectID)).
			AND(table.VaultItems.Environment.EQ(NewEnumValue(req.Environment))).
			AND(table.VaultItems.NameHash.IN(hashExpressions...)),
	)

	if err := stmt.Query(h.db, &items); err != nil {
		slog.Error("secrets.bulk_fetch: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch secrets"})
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
	Ciphertext string          `json:"ciphertext"` // base64
	Nonce      string          `json:"nonce"`      // base64
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// SecretMetadataMergeBody documents the JSON body for PATCH /secrets/{nameHash}/metadata.
// Allowed keys: mime_type, description, tags, labels. Unknown keys are rejected. JSON null removes a key.
type SecretMetadataMergeBody struct {
	MimeType    string            `json:"mime_type,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// @Summary		Update secret
// @Description	Replace ciphertext and nonce for an existing secret; the prior row is archived and the version counter increments. Optional `metadata` in the body is shallow-merged with existing server-side metadata (same validation as PATCH metadata). Use GET /{nameHash}/versions for history.
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			nameHash	path		string				true	"HMAC-SHA256 name hash (base64, URL-encoded)"
// @Param			project_id	query		string				true	"Project ID"
// @Param			environment	query		string				true	"Environment"
// @Param			body		body		UpdateSecretRequest	true	"New ciphertext, nonce, and optional metadata merge"
// @Success		200			{object}	SecretResponse
// @Failure		400			{object}	ErrorResponse	"Missing params or invalid base64"
// @Failure		404			{object}	ErrorResponse	"Secret not found"
// @Failure		500			{object}	ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash} [put]
// @Router			/secrets/{nameHash} [put]
func (h *SecretsHandler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid name_hash in URL"})
		return
	}

	var req UpdateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Ciphertext == "" || req.Nonce == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ciphertext and nonce are required"})
		return
	}

	ciphertext, err := base64.StdEncoding.DecodeString(req.Ciphertext)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in ciphertext"})
		return
	}
	nonce, err := base64.StdEncoding.DecodeString(req.Nonce)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in nonce"})
		return
	}

	now := time.Now().UTC()

	var current model.VaultItems
	fetchCurrent := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(NewEnumValue(env))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)
	if err := fetchCurrent.Query(h.db, &current); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "secret not found"})
			return
		}
		slog.Error("secrets.update: fetch current", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch secret"})
		return
	}

	var mergedMeta []byte
	if len(req.Metadata) > 0 {
		mergedMeta, err = mergeSecretMetadataJSON(current.Metadata, req.Metadata)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
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
		// Non-fatal — log and continue.
		slog.Warn("secrets.update: archive version failed", "error", err)
	}

	var execErr error
	if mergedMeta != nil {
		_, execErr = table.VaultItems.UPDATE(
			table.VaultItems.Ciphertext,
			table.VaultItems.Nonce,
			table.VaultItems.Version,
			table.VaultItems.Metadata,
			table.VaultItems.UpdatedAt,
		).SET(
			Bytea(ciphertext),
			Bytea(nonce),
			table.VaultItems.Version.ADD(Int(1)),
			String(string(mergedMeta)),
			TimestampzT(now),
		).WHERE(
			table.VaultItems.ID.EQ(UUID(current.ID)),
		).Exec(h.db)
	} else {
		_, execErr = table.VaultItems.UPDATE(
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
		).Exec(h.db)
	}

	if execErr != nil {
		slog.Error("secrets.update: exec", "error", execErr)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update secret"})
		return
	}

	var item model.VaultItems
	fetchStmt := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(table.VaultItems.ID.EQ(UUID(current.ID)))

	if err := fetchStmt.Query(h.db, &item); err != nil {
		slog.Error("secrets.update: fetch updated", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch updated secret"})
		return
	}

	writeJSON(w, http.StatusOK, toSecretResponse(item))
}

// --- Delete ---

// @Summary		Delete secret
// @Description	Permanently remove a secret and all its archived versions from the vault. This action is irreversible.
// @Tags			secrets
// @Param			nameHash	path	string	true	"HMAC-SHA256 name hash (base64, URL-encoded)"
// @Param			project_id	query	string	true	"Project ID"
// @Param			environment	query	string	true	"Environment"
// @Success		200		{object}	map[string]string	"status: deleted"
// @Failure		400		{object}	ErrorResponse		"Missing params or invalid name hash"
// @Failure		404		{object}	ErrorResponse		"Secret not found"
// @Failure		500		{object}	ErrorResponse		"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash} [delete]
// @Router			/secrets/{nameHash} [delete]
func (h *SecretsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	slog.Info("secrets.delete: raw path param", "nameHash", nameHashB64)
	nameHash, err := decodeNameHash(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid name_hash in URL"})
		return
	}

	deleteStmt := table.VaultItems.DELETE().WHERE(
		table.VaultItems.ProjectID.EQ(UUID(projectID)).
			AND(table.VaultItems.Environment.EQ(NewEnumValue(env))).
			AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
	)

	result, err := deleteStmt.Exec(h.db)
	if err != nil {
		slog.Error("secrets.delete: exec", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete secret"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "secret not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- List ---

type SecretListItem struct {
	ID          string            `json:"id"`
	NameHash    string            `json:"name_hash"`
	Environment model.Environment `json:"environment"`
	Version     int               `json:"version"`
	Metadata    json.RawMessage   `json:"metadata,omitempty"`
	UpdatedAt   string            `json:"updated_at"`
	CreatedAt   string            `json:"created_at"`
}

type ListSecretsResponse struct {
	Secrets []SecretListItem `json:"secrets"`
	Meta    Meta             `json:"meta"`
}

// @Summary		List secrets
// @Description	List secret rows for a project: name hash, environment, version, timestamps, and plaintext `metadata` (never ciphertext or nonces). Supports pagination, sorting, and filters by environment and version.
// @Tags			secrets
// @Produce		json
// @Param			project_id		query		string	true	"Project ID"
// @Param			environment		query		string	false	"Filter by environment (development | staging | production)"
// @Param			version			query		int		false	"Filter by exact version number"
// @Param			page			query		int		false	"Page number (default: 1)"
// @Param			per_page		query		int		false	"Items per page (default: 20, max: 100)"
// @Param			sort_by			query		string	false	"Sort field: updated_at | created_at | version | environment (default: updated_at)"
// @Param			sort_dir		query		string	false	"Sort direction: asc | desc (default: desc)"
// @Success		200				{object}	ListSecretsResponse
// @Failure		400				{object}	ErrorResponse	"Missing project_id"
// @Failure		500				{object}	ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets [get]
// @Router			/secrets [get]
func (h *SecretsHandler) List(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "project_id query param is required"})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project_id"})
		return
	}

	params := ParseListParams(r)
	environment := r.URL.Query().Get("environment")
	versionStr := r.URL.Query().Get("version")

	condition := table.VaultItems.ProjectID.EQ(UUID(projectID))

	if environment != "" {
		condition = condition.AND(table.VaultItems.Environment.EQ(NewEnumValue(environment)))
	}

	if versionStr != "" {
		var v int64
		if _, err := fmt.Sscanf(versionStr, "%d", &v); err == nil {
			condition = condition.AND(table.VaultItems.Version.EQ(Int(v)))
		}
	}

	// Count total matching rows.
	var countResult struct {
		Count int64 `alias:"count"`
	}
	countStmt := SELECT(COUNT(table.VaultItems.ID).AS("count")).
		FROM(table.VaultItems).
		WHERE(condition)
	_ = countStmt.Query(h.db, &countResult)

	// Sort mapping.
	var orderBy OrderByClause
	switch params.SortBy {
	case "created_at":
		if params.SortDir == "asc" {
			orderBy = table.VaultItems.CreatedAt.ASC()
		} else {
			orderBy = table.VaultItems.CreatedAt.DESC()
		}
	case "version":
		if params.SortDir == "asc" {
			orderBy = table.VaultItems.Version.ASC()
		} else {
			orderBy = table.VaultItems.Version.DESC()
		}
	case "environment":
		if params.SortDir == "asc" {
			orderBy = table.VaultItems.Environment.ASC()
		} else {
			orderBy = table.VaultItems.Environment.DESC()
		}
	default: // updated_at
		if params.SortDir == "asc" {
			orderBy = table.VaultItems.UpdatedAt.ASC()
		} else {
			orderBy = table.VaultItems.UpdatedAt.DESC()
		}
	}

	var items []model.VaultItems
	stmt := SELECT(
		table.VaultItems.ID,
		table.VaultItems.NameHash,
		table.VaultItems.Environment,
		table.VaultItems.Version,
		table.VaultItems.Metadata,
		table.VaultItems.CreatedAt,
		table.VaultItems.UpdatedAt,
	).FROM(table.VaultItems).
		WHERE(condition).
		ORDER_BY(orderBy).
		LIMIT(params.Limit()).
		OFFSET(params.Offset())

	if err := stmt.Query(h.db, &items); err != nil && !errors.Is(err, qrm.ErrNoRows) {
		slog.Error("secrets.list: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list secrets"})
		return
	}

	result := make([]SecretListItem, 0, len(items))
	for _, item := range items {
		md := json.RawMessage(`{}`)
		if item.Metadata != "" {
			md = json.RawMessage(item.Metadata)
		}
		result = append(result, SecretListItem{
			ID:          item.ID.String(),
			NameHash:    base64.StdEncoding.EncodeToString(item.NameHash),
			Environment: item.Environment,
			Version:     int(item.Version),
			Metadata:    md,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, ListSecretsResponse{
		Secrets: result,
		Meta:    NewMeta(int(countResult.Count), params.Page, params.PerPage),
	})
}

// --- Versions ---

type VersionItem struct {
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
}

type VersionsResponse struct {
	CurrentVersion int           `json:"current_version"`
	Versions       []VersionItem `json:"versions"`
}

// @Summary		List secret versions
// @Description	Return the full version history for a secret. The current version is shown separately from the archived versions. Versions are ordered newest first.
// @Tags			secrets
// @Produce		json
// @Param			nameHash	path		string	true	"HMAC-SHA256 name hash (base64, URL-encoded)"
// @Param			project_id	query		string	true	"Project ID"
// @Param			environment	query		string	true	"Environment"
// @Success		200			{object}	VersionsResponse
// @Failure		400			{object}	ErrorResponse	"Missing params or invalid name hash"
// @Failure		404			{object}	ErrorResponse	"Secret not found"
// @Failure		500			{object}	ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash}/versions [get]
// @Router			/secrets/{nameHash}/versions [get]
func (h *SecretsHandler) Versions(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid name_hash in URL"})
		return
	}

	var current model.VaultItems
	currentStmt := SELECT(table.VaultItems.ID, table.VaultItems.Version).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(NewEnumValue(env))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)

	if err := currentStmt.Query(h.db, &current); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "secret not found"})
			return
		}
		slog.Error("secrets.versions: fetch current", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch secret"})
		return
	}

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
		CurrentVersion: int(current.Version),
		Versions:       versions,
	})
}

// --- Rollback ---

type RollbackRequest struct {
	Version int `json:"version"`
}

// @Summary		Rollback secret
// @Description	Revert a secret to a previously archived version. The current ciphertext is archived first, then the target version's ciphertext is restored. The version counter continues incrementing — it is never reset. Returns the updated secret after rollback.
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			nameHash	path		string			true	"HMAC-SHA256 name hash (base64, URL-encoded)"
// @Param			project_id	query		string			true	"Project ID"
// @Param			environment	query		string			true	"Environment"
// @Param			body		body		RollbackRequest	true	"Target version number"
// @Success		200			{object}	SecretResponse
// @Failure		400			{object}	ErrorResponse	"Missing params, invalid name hash, or missing version in body"
// @Failure		404			{object}	ErrorResponse	"Secret not found, or target version not found in archive"
// @Failure		500			{object}	ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash}/rollback [post]
// @Router			/secrets/{nameHash}/rollback [post]
func (h *SecretsHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid name_hash in URL"})
		return
	}

	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Version <= 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "version must be a positive integer"})
		return
	}

	var current model.VaultItems
	currentStmt := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(NewEnumValue(env))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		)

	if err := currentStmt.Query(h.db, &current); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "secret not found"})
			return
		}
		slog.Error("secrets.rollback: fetch current", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch secret"})
		return
	}

	var target model.VaultItemVersions
	targetStmt := SELECT(table.VaultItemVersions.AllColumns).
		FROM(table.VaultItemVersions).
		WHERE(
			table.VaultItemVersions.ItemID.EQ(UUID(current.ID)).
				AND(table.VaultItemVersions.Version.EQ(Int(int64(req.Version)))),
		)

	if err := targetStmt.Query(h.db, &target); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error: fmt.Sprintf("version %d not found in archive", req.Version),
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
		slog.Warn("secrets.rollback: archive current failed", "error", err)
	}

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
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "rollback failed"})
		return
	}

	var item model.VaultItems
	fetchStmt := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(table.VaultItems.ID.EQ(UUID(current.ID)))

	if err := fetchStmt.Query(h.db, &item); err != nil {
		slog.Error("secrets.rollback: fetch updated", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "rollback succeeded but failed to fetch updated secret"})
		return
	}

	writeJSON(w, http.StatusOK, toSecretResponse(item))
}

// @Summary		Patch secret metadata
// @Description	Shallow-merge plaintext metadata for a vault item without changing ciphertext, nonce, or the secret version counter. Use this when only labels, MIME hints, or descriptions change. Body must be a JSON object; unknown keys return 400. Send JSON null for a key to remove it. Total metadata size is capped server-side.
// @Tags			secrets
// @Accept			json
// @Produce		json
// @Param			nameHash	path		string					true	"HMAC-SHA256 name hash (base64, URL-encoded)"
// @Param			project_id	query		string					true	"Project ID"
// @Param			environment	query		string					true	"Environment (development | staging | production)"
// @Param			body		body		SecretMetadataMergeBody	true	"Fields to merge (partial object)"
// @Success		200			{object}	SecretResponse
// @Failure		400			{object}	ErrorResponse	"Empty body, invalid JSON, unknown keys, or metadata limits exceeded"
// @Failure		404			{object}	ErrorResponse	"Secret not found"
// @Failure		500			{object}	ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/sdk/secrets/{nameHash}/metadata [patch]
// @Router			/secrets/{nameHash}/metadata [patch]
func (h *SecretsHandler) PatchMetadata(w http.ResponseWriter, r *http.Request) {
	projectID, env, err := parseProjectEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	nameHashB64 := chi.URLParam(r, "nameHash")
	nameHash, err := decodeNameHash(nameHashB64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid name_hash in URL"})
		return
	}

	patch, err := io.ReadAll(io.LimitReader(r.Body, maxSecretMetadataBytes+4096))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	if len(patch) == 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "empty body"})
		return
	}
	if !json.Valid(patch) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON"})
		return
	}

	var current model.VaultItems
	if err := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(
			table.VaultItems.ProjectID.EQ(UUID(projectID)).
				AND(table.VaultItems.Environment.EQ(NewEnumValue(env))).
				AND(table.VaultItems.NameHash.EQ(Bytea(nameHash))),
		).Query(h.db, &current); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "secret not found"})
			return
		}
		slog.Error("secrets.patch_meta: fetch", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch secret"})
		return
	}

	out, err := mergeSecretMetadataJSON(current.Metadata, patch)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	now := time.Now().UTC()
	if _, err := table.VaultItems.UPDATE(
		table.VaultItems.Metadata,
		table.VaultItems.UpdatedAt,
	).SET(
		String(string(out)),
		TimestampzT(now),
	).WHERE(table.VaultItems.ID.EQ(UUID(current.ID))).
		Exec(h.db); err != nil {
		slog.Error("secrets.patch_meta: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update metadata"})
		return
	}

	var item model.VaultItems
	if err := SELECT(table.VaultItems.AllColumns).
		FROM(table.VaultItems).
		WHERE(table.VaultItems.ID.EQ(UUID(current.ID))).
		Query(h.db, &item); err != nil {
		slog.Error("secrets.patch_meta: refetch", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to load secret"})
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
	if decoded, err := url.PathUnescape(s); err == nil {
		s = decoded
	}

	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}

func toSecretResponse(item model.VaultItems) SecretResponse {
	var md json.RawMessage
	if item.Metadata != "" {
		md = json.RawMessage(item.Metadata)
	} else {
		md = json.RawMessage(`{}`)
	}
	return SecretResponse{
		ID:          item.ID.String(),
		ProjectID:   item.ProjectID.String(),
		Environment: item.Environment,
		NameHash:    base64.StdEncoding.EncodeToString(item.NameHash),
		Ciphertext:  base64.StdEncoding.EncodeToString(item.Ciphertext),
		Nonce:       base64.StdEncoding.EncodeToString(item.Nonce),
		Metadata:    md,
		Version:     int(item.Version),
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}
}
