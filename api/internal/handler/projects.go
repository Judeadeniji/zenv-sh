package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// ProjectsHandler handles project CRUD and project crypto endpoints.
type ProjectsHandler struct {
	db *sql.DB
}

func NewProjectsHandler(db *sql.DB) *ProjectsHandler {
	return &ProjectsHandler{db: db}
}

// --- Create Project ---

type CreateProjectRequest struct {
	OrganizationID         string `json:"organization_id"`
	Name                   string `json:"name"`
	ProjectSalt            string `json:"project_salt"`              // base64 — generated client-side
	WrappedProjectDEK      string `json:"wrapped_project_dek"`       // base64 — Project DEK wrapped with Project KEK
	WrappedProjectVaultKey string `json:"wrapped_project_vault_key"` // base64 — Project Vault Key wrapped with user's public key
}

type ProjectResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	CreatedAt      string `json:"created_at"`
}

// @Summary		Create project
// @Description	Create a project with client-generated crypto material. Project Vault Key shown once at creation, never stored on server.
// @Tags			projects
// @Accept			json
// @Produce		json
// @Param			body	body		CreateProjectRequest	true	"Project config + crypto material"
// @Success		201		{object}	ProjectResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/projects [post]
func (h *ProjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.OrganizationID == "" || req.Name == "" || req.ProjectSalt == "" || req.WrappedProjectDEK == "" || req.WrappedProjectVaultKey == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "all fields are required"})
		return
	}

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid organization_id"})
		return
	}
	userID, err := uuid.Parse(sess.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "invalid session"})
		return
	}

	projectSalt, err := base64.StdEncoding.DecodeString(req.ProjectSalt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in project_salt"})
		return
	}
	wrappedProjectDEK, err := base64.StdEncoding.DecodeString(req.WrappedProjectDEK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in wrapped_project_dek"})
		return
	}
	wrappedProjectVaultKey, err := base64.StdEncoding.DecodeString(req.WrappedProjectVaultKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in wrapped_project_vault_key"})
		return
	}

	// Use a transaction — project + vault key + grant must all succeed together.
	tx, err := h.db.Begin()
	if err != nil {
		slog.Error("create project: begin tx", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}
	defer tx.Rollback()

	projectID := uuid.New()
	now := time.Now().UTC()

	// 1. Insert project
	insertProject := table.Projects.INSERT(
		table.Projects.ID,
		table.Projects.OrganizationID,
		table.Projects.Name,
		table.Projects.CreatedAt,
	).VALUES(projectID, orgID, req.Name, now)

	_, err = insertProject.Exec(tx)
	if err != nil {
		// Check for unique constraint violation
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "project name already exists in this organization"})
		return
	}

	// 2. Insert project vault key
	insertVaultKey := table.ProjectVaultKeys.INSERT(
		table.ProjectVaultKeys.ID,
		table.ProjectVaultKeys.ProjectID,
		table.ProjectVaultKeys.ProjectSalt,
		table.ProjectVaultKeys.WrappedProjectDek,
		table.ProjectVaultKeys.CreatedAt,
	).VALUES(uuid.New(), projectID, projectSalt, wrappedProjectDEK, now)

	_, err = insertVaultKey.Exec(tx)
	if err != nil {
		slog.Error("create project: insert vault key", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create project crypto"})
		return
	}

	// 3. Insert key grant for the creating user
	insertGrant := table.ProjectKeyGrants.INSERT(
		table.ProjectKeyGrants.ID,
		table.ProjectKeyGrants.ProjectID,
		table.ProjectKeyGrants.UserID,
		table.ProjectKeyGrants.WrappedProjectVaultKey,
		table.ProjectKeyGrants.GrantedAt,
	).VALUES(uuid.New(), projectID, userID, wrappedProjectVaultKey, now)

	_, err = insertGrant.Exec(tx)
	if err != nil {
		slog.Error("create project: insert key grant", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create key grant"})
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("create project: commit", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to commit"})
		return
	}

	writeJSON(w, http.StatusCreated, ProjectResponse{
		ID:             projectID.String(),
		OrganizationID: orgID.String(),
		Name:           req.Name,
		CreatedAt:      now.Format(time.RFC3339),
	})
}

// --- List Projects ---

type ListProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Meta     Meta              `json:"meta"`
}

// @Summary		List projects
// @Description	List all projects in an organization.
// @Tags			projects
// @Produce		json
// @Param			organization_id	query		string	true	"Organization ID"
// @Param			page			query		int		false	"Page number"
// @Param			per_page		query		int		false	"Items per page"
// @Param			sort_by			query		string	false	"Sort by field"
// @Param			sort_dir		query		string	false	"Sort direction (asc/desc)"
// @Param			search			query		string	false	"Search by project name"
// @Success		200				{object}	ListProjectsResponse
// @Security		SessionAuth
// @Router			/projects [get]
func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	orgIDStr := r.URL.Query().Get("organization_id")
	if orgIDStr == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "organization_id query param is required"})
		return
	}

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid organization_id"})
		return
	}

	params := ParseListParams(r)

	condition := table.Projects.OrganizationID.EQ(UUID(orgID))
	if params.Search != "" {
		condition = condition.AND(LOWER(table.Projects.Name).LIKE(String("%" + strings.ToLower(params.Search) + "%")))
	}

	var countResult struct {
		Count int64 `alias:"count"`
	}
	countStmt := SELECT(COUNT(table.Projects.ID).AS("count")).
		FROM(table.Projects).
		WHERE(condition)
	_ = countStmt.Query(h.db, &countResult)

	var orderBy OrderByClause
	switch params.SortBy {
	case "name":
		if params.SortDir == "asc" {
			orderBy = table.Projects.Name.ASC()
		} else {
			orderBy = table.Projects.Name.DESC()
		}
	default: // "created_at"
		if params.SortDir == "asc" {
			orderBy = table.Projects.CreatedAt.ASC()
		} else {
			orderBy = table.Projects.CreatedAt.DESC()
		}
	}

	var projects []model.Projects
	stmt := SELECT(
		table.Projects.ID,
		table.Projects.OrganizationID,
		table.Projects.Name,
		table.Projects.CreatedAt,
	).FROM(table.Projects).
		WHERE(condition).
		ORDER_BY(orderBy).
		LIMIT(params.Limit()).
		OFFSET(params.Offset())

	err = stmt.Query(h.db, &projects)
	if err != nil && !errors.Is(err, qrm.ErrNoRows) {
		slog.Error("projects.list: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list projects"})
		return
	}

	resp := ListProjectsResponse{
		Projects: make([]ProjectResponse, 0, len(projects)),
		Meta:     NewMeta(int(countResult.Count), params.Page, params.PerPage),
	}
	for _, p := range projects {
		resp.Projects = append(resp.Projects, ProjectResponse{
			ID:             p.ID.String(),
			OrganizationID: p.OrganizationID.String(),
			Name:           p.Name,
			CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Get Project ---

// @Summary		Get project
// @Description	Get a single project by ID.
// @Tags			projects
// @Produce		json
// @Param			projectID	path		string	true	"Project UUID"
// @Success		200			{object}	ProjectResponse
// @Failure		404			{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/projects/{projectID} [get]
func (h *ProjectsHandler) Get(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}

	var project model.Projects
	stmt := SELECT(
		table.Projects.ID,
		table.Projects.OrganizationID,
		table.Projects.Name,
		table.Projects.CreatedAt,
	).FROM(table.Projects).WHERE(table.Projects.ID.EQ(UUID(projectID)))

	err = stmt.Query(h.db, &project)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "project not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get project"})
		return
	}

	writeJSON(w, http.StatusOK, ProjectResponse{
		ID:             project.ID.String(),
		OrganizationID: project.OrganizationID.String(),
		Name:           project.Name,
		CreatedAt:      project.CreatedAt.Format(time.RFC3339),
	})
}

// --- Delete Project ---

// @Summary		Delete project
// @Description	Delete a project and all associated data (secrets, tokens, key grants).
// @Tags			projects
// @Produce		json
// @Param			projectID	path		string	true	"Project UUID"
// @Success		200			{object}	map[string]string
// @Failure		404			{object}	ErrorResponse
// @Failure		500			{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/projects/{projectID} [delete]
func (h *ProjectsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}

	var project model.Projects
	checkStmt := SELECT(table.Projects.ID).FROM(table.Projects).WHERE(table.Projects.ID.EQ(UUID(projectID)))
	if err := checkStmt.Query(h.db, &project); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "project not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to check project"})
		return
	}

	deleteStmt := table.Projects.DELETE().WHERE(table.Projects.ID.EQ(UUID(projectID)))
	if _, err = deleteStmt.Exec(h.db); err != nil {
		slog.Error("delete project", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete project"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"deleted": projectID.String()})
}

// --- Rotation: Two-Phase Commit Re-encryption ---

type StartRotationRequest struct {
	TotalItems int `json:"total_items"`
}

type StartRotationResponse struct {
	RotationID string `json:"rotation_id"`
	Status     string `json:"status"`
}

// StartRotation initiates a DEK rotation for a project.
//
//	@Summary		Start DEK rotation
//	@Description	Initiates a two-phase DEK rotation. Returns a rotation_id for staging and committing.
//	@Tags			rotation
//	@Accept			json
//	@Produce		json
//	@Param			projectID	path		string					true	"Project UUID"
//	@Param			body		body		StartRotationRequest	true	"Rotation parameters"
//	@Success		201			{object}	StartRotationResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/projects/{projectID}/rotation/start [post]
func (h *ProjectsHandler) StartRotation(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.UserID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var req StartRotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TotalItems <= 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "total_items is required and must be positive"})
		return
	}

	// Check no active rotation exists for this project.
	var existing int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM project_rotations WHERE project_id = $1 AND status IN ('staging', 'committing')`,
		projectID,
	).Scan(&existing)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to check rotations"})
		return
	}
	if existing > 0 {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "a rotation is already in progress for this project"})
		return
	}

	rotationID := uuid.New()
	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO project_rotations (project_id, rotation_id, total_items, initiated_by)
		 VALUES ($1, $2, $3, $4)`,
		projectID, rotationID, req.TotalItems, sess.UserID,
	)
	if err != nil {
		slog.Error("start rotation: insert", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to start rotation"})
		return
	}

	writeJSON(w, http.StatusCreated, StartRotationResponse{
		RotationID: rotationID.String(),
		Status:     "staging",
	})
}

type StageItem struct {
	VaultItemID   string `json:"vault_item_id"`
	NewCiphertext string `json:"new_ciphertext"` // base64
	NewNonce      string `json:"new_nonce"`      // base64
}

type StageRequest struct {
	Items []StageItem `json:"items"`
}

type StageResponse struct {
	Staged      int `json:"staged"`
	TotalStaged int `json:"total_staged"`
	Total       int `json:"total"`
}

// StageRotation stages a batch of re-encrypted items (Phase 1).
//
//	@Summary		Stage re-encrypted items
//	@Description	Uploads a batch of re-encrypted ciphertexts to the staging table. Call repeatedly until all items are staged.
//	@Tags			rotation
//	@Accept			json
//	@Produce		json
//	@Param			projectID	path		string			true	"Project UUID"
//	@Param			rotationID	path		string			true	"Rotation UUID"
//	@Param			body		body		StageRequest	true	"Batch of re-encrypted items"
//	@Success		200			{object}	StageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/projects/{projectID}/rotation/{rotationID}/stage [post]
func (h *ProjectsHandler) StageRotation(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}
	rotationID, err := uuid.Parse(chi.URLParam(r, "rotationID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid rotation ID"})
		return
	}

	var req StageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "items array is required"})
		return
	}

	// Verify rotation exists and is in staging state.
	var status string
	var totalItems int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT status, total_items FROM project_rotations WHERE rotation_id = $1 AND project_id = $2`,
		rotationID, projectID,
	).Scan(&status, &totalItems)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "rotation not found"})
		return
	}
	if status != "staging" {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "rotation is not in staging state"})
		return
	}

	// Bulk insert staged items.
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	for _, item := range req.Items {
		itemID, err := uuid.Parse(item.VaultItemID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid vault_item_id: " + item.VaultItemID})
			return
		}
		ct, err := base64.StdEncoding.DecodeString(item.NewCiphertext)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 ciphertext"})
			return
		}
		nonce, err := base64.StdEncoding.DecodeString(item.NewNonce)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 nonce"})
			return
		}

		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO vault_item_rotations (project_id, rotation_id, vault_item_id, new_ciphertext, new_nonce)
			 VALUES ($1, $2, $3, $4, $5)`,
			projectID, rotationID, itemID, ct, nonce,
		)
		if err != nil {
			slog.Error("stage rotation: insert item", "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to stage item"})
			return
		}
	}

	// Update staged count.
	var totalStaged int
	err = tx.QueryRowContext(r.Context(),
		`UPDATE project_rotations SET staged_items = staged_items + $1
		 WHERE rotation_id = $2 RETURNING staged_items`,
		len(req.Items), rotationID,
	).Scan(&totalStaged)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update staged count"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to commit staging"})
		return
	}

	writeJSON(w, http.StatusOK, StageResponse{
		Staged:      len(req.Items),
		TotalStaged: totalStaged,
		Total:       totalItems,
	})
}

type CommitRotationRequest struct {
	NewWrappedProjectDEK string `json:"new_wrapped_project_dek"` // base64
	NewProjectSalt       string `json:"new_project_salt"`        // base64
	NewKeyGrants         []struct {
		UserID                 string `json:"user_id"`
		WrappedProjectVaultKey string `json:"wrapped_project_vault_key"` // base64
	} `json:"new_key_grants"`
}

// CommitRotation atomically applies all staged re-encrypted items (Phase 2).
//
//	@Summary		Commit DEK rotation
//	@Description	Atomically applies all staged ciphertexts, updates the wrapped DEK and key grants. All-or-nothing via Postgres transaction.
//	@Tags			rotation
//	@Accept			json
//	@Produce		json
//	@Param			projectID	path		string					true	"Project UUID"
//	@Param			rotationID	path		string					true	"Rotation UUID"
//	@Param			body		body		CommitRotationRequest	true	"New wrapped DEK, salt, and key grants"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/projects/{projectID}/rotation/{rotationID}/commit [post]
func (h *ProjectsHandler) CommitRotation(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}
	rotationID, err := uuid.Parse(chi.URLParam(r, "rotationID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid rotation ID"})
		return
	}

	var req CommitRotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	newWrappedDEK, err := base64.StdEncoding.DecodeString(req.NewWrappedProjectDEK)
	if err != nil || len(newWrappedDEK) == 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid new_wrapped_project_dek"})
		return
	}
	newSalt, err := base64.StdEncoding.DecodeString(req.NewProjectSalt)
	if err != nil || len(newSalt) == 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid new_project_salt"})
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	// 1. Set rotation status to committing.
	var status string
	err = tx.QueryRowContext(r.Context(),
		`UPDATE project_rotations SET status = 'committing' WHERE rotation_id = $1 AND status = 'staging' RETURNING status`,
		rotationID,
	).Scan(&status)
	if err != nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "rotation not in staging state"})
		return
	}

	// 2. Apply staged ciphertexts to vault_items.
	_, err = tx.ExecContext(r.Context(),
		`UPDATE vault_items SET
			ciphertext = r.new_ciphertext,
			nonce = r.new_nonce,
			dek_version = vault_items.dek_version + 1,
			version = vault_items.version + 1,
			updated_at = NOW()
		 FROM vault_item_rotations r
		 WHERE r.rotation_id = $1
		   AND vault_items.id = r.vault_item_id`,
		rotationID,
	)
	if err != nil {
		slog.Error("commit rotation: update items", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to apply rotation"})
		return
	}

	// 3. Update project vault keys.
	_, err = tx.ExecContext(r.Context(),
		`UPDATE project_vault_keys SET
			wrapped_project_dek = $1,
			project_salt = $2,
			dek_version = dek_version + 1
		 WHERE project_id = $3`,
		newWrappedDEK, newSalt, projectID,
	)
	if err != nil {
		slog.Error("commit rotation: update vault keys", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update vault keys"})
		return
	}

	// 4. Update key grants.
	for _, grant := range req.NewKeyGrants {
		grantUserID, err := uuid.Parse(grant.UserID)
		if err != nil {
			continue
		}
		wrappedKey, err := base64.StdEncoding.DecodeString(grant.WrappedProjectVaultKey)
		if err != nil {
			continue
		}
		_, err = tx.ExecContext(r.Context(),
			`UPDATE project_key_grants SET wrapped_project_vault_key = $1
			 WHERE project_id = $2 AND user_id = $3`,
			wrappedKey, projectID, grantUserID,
		)
		if err != nil {
			slog.Error("commit rotation: update grant", "user_id", grantUserID, "error", err)
		}
	}

	// 5. Clean up staging rows.
	_, err = tx.ExecContext(r.Context(),
		`DELETE FROM vault_item_rotations WHERE rotation_id = $1`, rotationID,
	)
	if err != nil {
		slog.Error("commit rotation: cleanup staging", "error", err)
	}

	// 6. Mark rotation complete.
	_, err = tx.ExecContext(r.Context(),
		`UPDATE project_rotations SET status = 'complete', completed_at = NOW() WHERE rotation_id = $1`,
		rotationID,
	)
	if err != nil {
		slog.Error("commit rotation: mark complete", "error", err)
	}

	if err := tx.Commit(); err != nil {
		slog.Error("commit rotation: commit tx", "error", err)
		// Mark as failed.
		h.db.ExecContext(r.Context(),
			`UPDATE project_rotations SET status = 'failed' WHERE rotation_id = $1`, rotationID,
		)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to commit rotation"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "complete", "rotation_id": rotationID.String()})
}

// CancelRotation cleans up a staging or failed rotation.
//
//	@Summary		Cancel rotation
//	@Description	Deletes staging rows and the rotation record. Only works for rotations in 'staging' or 'failed' state.
//	@Tags			rotation
//	@Produce		json
//	@Param			projectID	path		string	true	"Project UUID"
//	@Param			rotationID	path		string	true	"Rotation UUID"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/projects/{projectID}/rotation/{rotationID} [delete]
func (h *ProjectsHandler) CancelRotation(w http.ResponseWriter, r *http.Request) {
	rotationID, err := uuid.Parse(chi.URLParam(r, "rotationID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid rotation ID"})
		return
	}

	_, err = h.db.ExecContext(r.Context(),
		`DELETE FROM project_rotations WHERE rotation_id = $1 AND status IN ('staging', 'failed')`,
		rotationID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to cancel rotation"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"cancelled": rotationID.String()})
}

// --- SDK Project Create ---

// CreateForToken creates a project on behalf of the token's creator.
func (h *ProjectsHandler) CreateForToken(w http.ResponseWriter, r *http.Request) {
	userID, err := tokenCreatorID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.OrganizationID == "" || req.Name == "" || req.ProjectSalt == "" || req.WrappedProjectDEK == "" || req.WrappedProjectVaultKey == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "all fields are required"})
		return
	}

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid organization_id"})
		return
	}

	projectSalt, err := base64.StdEncoding.DecodeString(req.ProjectSalt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in project_salt"})
		return
	}
	wrappedProjectDEK, err := base64.StdEncoding.DecodeString(req.WrappedProjectDEK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in wrapped_project_dek"})
		return
	}
	wrappedProjectVaultKey, err := base64.StdEncoding.DecodeString(req.WrappedProjectVaultKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in wrapped_project_vault_key"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		slog.Error("create project for token: begin tx", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}
	defer tx.Rollback()

	projectID := uuid.New()
	now := time.Now().UTC()

	insertProject := table.Projects.INSERT(
		table.Projects.ID,
		table.Projects.OrganizationID,
		table.Projects.Name,
		table.Projects.CreatedAt,
	).VALUES(projectID, orgID, req.Name, now)

	if _, err = insertProject.Exec(tx); err != nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "project name already exists in this organization"})
		return
	}

	insertVaultKey := table.ProjectVaultKeys.INSERT(
		table.ProjectVaultKeys.ID,
		table.ProjectVaultKeys.ProjectID,
		table.ProjectVaultKeys.ProjectSalt,
		table.ProjectVaultKeys.WrappedProjectDek,
		table.ProjectVaultKeys.CreatedAt,
	).VALUES(uuid.New(), projectID, projectSalt, wrappedProjectDEK, now)

	if _, err = insertVaultKey.Exec(tx); err != nil {
		slog.Error("create project for token: insert vault key", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create project crypto"})
		return
	}

	insertGrant := table.ProjectKeyGrants.INSERT(
		table.ProjectKeyGrants.ID,
		table.ProjectKeyGrants.ProjectID,
		table.ProjectKeyGrants.UserID,
		table.ProjectKeyGrants.WrappedProjectVaultKey,
		table.ProjectKeyGrants.GrantedAt,
	).VALUES(uuid.New(), projectID, userID, wrappedProjectVaultKey, now)

	if _, err = insertGrant.Exec(tx); err != nil {
		slog.Error("create project for token: insert key grant", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create key grant"})
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("create project for token: commit", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to commit"})
		return
	}

	writeJSON(w, http.StatusCreated, ProjectResponse{
		ID:             projectID.String(),
		OrganizationID: orgID.String(),
		Name:           req.Name,
		CreatedAt:      now.Format(time.RFC3339),
	})
}

// --- Project Crypto (SDK machine access) ---

type ProjectCryptoResponse struct {
	ProjectSalt       string `json:"project_salt"`             // base64
	WrappedProjectDEK string `json:"wrapped_project_dek"`      // base64
	VaultKeyType      string `json:"vault_key_type,omitempty"` // "pin" or "passphrase"
}

// @Summary		Get project crypto
// @Description	Returns project salt and wrapped Project DEK. SDK uses this to derive Project KEK and unwrap DEK.
// @Tags			projects
// @Produce		json
// @Param			projectID	path		string	true	"Project UUID"
// @Success		200			{object}	ProjectCryptoResponse
// @Failure		404			{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/projects/{projectID}/crypto [get]
// @Router			/sdk/projects/{projectID}/crypto [get]
func (h *ProjectsHandler) GetCrypto(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}

	var vk model.ProjectVaultKeys
	stmt := SELECT(
		table.ProjectVaultKeys.ProjectSalt,
		table.ProjectVaultKeys.WrappedProjectDek,
	).FROM(table.ProjectVaultKeys).WHERE(
		table.ProjectVaultKeys.ProjectID.EQ(UUID(projectID)),
	)

	err = stmt.Query(h.db, &vk)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "project crypto not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get project crypto"})
		return
	}

	resp := ProjectCryptoResponse{
		ProjectSalt:       base64.StdEncoding.EncodeToString(vk.ProjectSalt),
		WrappedProjectDEK: base64.StdEncoding.EncodeToString(vk.WrappedProjectDek),
	}

	// Resolve vault key type from token creator
	info := middleware.GetTokenInfo(r.Context())
	if info != nil && info.CreatedBy != "" {
		creatorID, _ := uuid.Parse(info.CreatedBy)
		var user model.Users
		userStmt := SELECT(table.Users.VaultKeyType).FROM(table.Users).WHERE(
			table.Users.ID.EQ(UUID(creatorID)),
		)
		if err := userStmt.Query(h.db, &user); err == nil {
			resp.VaultKeyType = user.VaultKeyType
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Key Grant ---

type KeyGrantResponse struct {
	WrappedProjectVaultKey string `json:"wrapped_project_vault_key"` // base64
}

// @Summary		Get project key grant
// @Description	Returns the current user's wrapped Project Vault Key for a project.
// @Tags			projects
// @Produce		json
// @Param			projectID	path		string	true	"Project UUID"
// @Success		200			{object}	KeyGrantResponse
// @Failure		404			{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/projects/{projectID}/key-grant [get]
func (h *ProjectsHandler) GetKeyGrant(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}

	userID, err := uuid.Parse(sess.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "invalid session"})
		return
	}

	var grant model.ProjectKeyGrants
	stmt := SELECT(
		table.ProjectKeyGrants.WrappedProjectVaultKey,
	).FROM(table.ProjectKeyGrants).WHERE(
		table.ProjectKeyGrants.ProjectID.EQ(UUID(projectID)).
			AND(table.ProjectKeyGrants.UserID.EQ(UUID(userID))),
	)

	if err := stmt.Query(h.db, &grant); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "no key grant found for this project"})
			return
		}
		slog.Error("projects.key_grant: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch key grant"})
		return
	}

	writeJSON(w, http.StatusOK, KeyGrantResponse{
		WrappedProjectVaultKey: base64.StdEncoding.EncodeToString(grant.WrappedProjectVaultKey),
	})
}

// --- List Key Grants (for rotation) ---

type KeyGrantMember struct {
	UserID    string `json:"user_id"`
	PublicKey string `json:"public_key"` // base64
}

type ListKeyGrantsResponse struct {
	Members []KeyGrantMember `json:"members"`
}

// ListKeyGrants returns all members with key grants for a project, including public keys.
//
//	@Summary		List key grants
//	@Description	Returns all project members and their public keys. Used during DEK rotation to re-wrap the Project Vault Key for each member.
//	@Tags			rotation
//	@Produce		json
//	@Param			projectID	path		string	true	"Project UUID"
//	@Success		200			{object}	ListKeyGrantsResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/projects/{projectID}/key-grants [get]
func (h *ProjectsHandler) ListKeyGrants(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT pkg.user_id, u.public_key
		 FROM project_key_grants pkg
		 JOIN users u ON u.id = pkg.user_id
		 WHERE pkg.project_id = $1`,
		projectID,
	)
	if err != nil {
		slog.Error("list_key_grants: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list key grants"})
		return
	}
	defer rows.Close()

	var members []KeyGrantMember
	for rows.Next() {
		var userID uuid.UUID
		var publicKey []byte
		if err := rows.Scan(&userID, &publicKey); err != nil {
			continue
		}
		members = append(members, KeyGrantMember{
			UserID:    userID.String(),
			PublicKey: base64.StdEncoding.EncodeToString(publicKey),
		})
	}

	if members == nil {
		members = []KeyGrantMember{}
	}

	writeJSON(w, http.StatusOK, ListKeyGrantsResponse{Members: members})
}

// GetKeyGrantForToken returns the token creator's key grant for a project.
func (h *ProjectsHandler) GetKeyGrantForToken(w http.ResponseWriter, r *http.Request) {
	userID, err := tokenCreatorID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}

	var grant model.ProjectKeyGrants
	stmt := SELECT(
		table.ProjectKeyGrants.WrappedProjectVaultKey,
	).FROM(table.ProjectKeyGrants).WHERE(
		table.ProjectKeyGrants.ProjectID.EQ(UUID(projectID)).
			AND(table.ProjectKeyGrants.UserID.EQ(UUID(userID))),
	)

	if err := stmt.Query(h.db, &grant); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "no key grant found for this project"})
			return
		}
		slog.Error("projects.key_grant_for_token: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch key grant"})
		return
	}

	writeJSON(w, http.StatusOK, KeyGrantResponse{
		WrappedProjectVaultKey: base64.StdEncoding.EncodeToString(grant.WrappedProjectVaultKey),
	})
}

// --- SDK Vault Material ---

type VaultMaterialResponse struct {
	Salt              string `json:"salt"`                // base64
	VaultKeyType      string `json:"vault_key_type"`      // "pin" or "passphrase"
	WrappedDEK        string `json:"wrapped_dek"`         // base64
	WrappedPrivateKey string `json:"wrapped_private_key"` // base64
	PublicKey         string `json:"public_key"`          // base64
}

// @Summary		Get vault material
// @Description	Returns the token creator's vault crypto material for client-side key derivation.
// @Tags			sdk
// @Produce		json
// @Success		200	{object}	VaultMaterialResponse
// @Failure		404	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/sdk/vault [get]
func (h *ProjectsHandler) GetVaultMaterial(w http.ResponseWriter, r *http.Request) {
	userID, err := tokenCreatorID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	var user model.Users
	stmt := SELECT(
		table.Users.Salt,
		table.Users.VaultKeyType,
		table.Users.WrappedDek,
		table.Users.WrappedPrivateKey,
		table.Users.PublicKey,
	).FROM(table.Users).WHERE(table.Users.ID.EQ(UUID(userID)))

	if err := stmt.Query(h.db, &user); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
		slog.Error("vault_material: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch vault material"})
		return
	}

	writeJSON(w, http.StatusOK, VaultMaterialResponse{
		Salt:              base64.StdEncoding.EncodeToString(user.Salt),
		VaultKeyType:      user.VaultKeyType,
		WrappedDEK:        base64.StdEncoding.EncodeToString(user.WrappedDek),
		WrappedPrivateKey: base64.StdEncoding.EncodeToString(user.WrappedPrivateKey),
		PublicKey:         base64.StdEncoding.EncodeToString(user.PublicKey),
	})
}

// --- Stats ---

type ProjectStatsResponse struct {
	TotalSecrets       int            `json:"total_secrets"`
	SecretsByEnv       map[string]int `json:"secrets_by_env"`
	TotalServiceTokens int            `json:"total_service_tokens"`
	TotalAuditLogs     int            `json:"total_audit_logs"`
}

// @Summary		Get project stats
// @Description	Get summary statistics for a project (secrets, tokens, audit logs).
// @Tags			projects
// @Produce		json
// @Param			projectID	path		string	true	"Project ID"
// @Success		200			{object}	ProjectStatsResponse
// @Failure		400			{object}	ErrorResponse
// @Failure		500			{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/projects/{projectID}/stats [get]
func (h *ProjectsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid project ID"})
		return
	}

	// 1. Secrets count by environment
	secretsStmt := SELECT(
		table.VaultItems.Environment,
		COUNT(table.VaultItems.ID).AS("count"),
	).FROM(table.VaultItems).
		WHERE(table.VaultItems.ProjectID.EQ(UUID(projectID))).
		GROUP_BY(table.VaultItems.Environment)

	var secretCounts []struct {
		Environment string `alias:"vault_items.environment"`
		Count       int    `alias:"count"`
	}
	if err := secretsStmt.Query(h.db, &secretCounts); err != nil {
		slog.Error("get project stats: secrets count", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch project stats"})
		return
	}

	totalSecrets := 0
	secretsByEnv := make(map[string]int)
	for _, sc := range secretCounts {
		secretsByEnv[sc.Environment] = sc.Count
		totalSecrets += sc.Count
	}

	// 2. Tokens count
	var tokensCount struct {
		Count int `alias:"count"`
	}
	tokensStmt := SELECT(COUNT(table.ServiceTokens.ID).AS("count")).
		FROM(table.ServiceTokens).
		WHERE(table.ServiceTokens.ProjectID.EQ(UUID(projectID)))
	if err := tokensStmt.Query(h.db, &tokensCount); err != nil {
		slog.Error("get project stats: tokens count", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch project stats"})
		return
	}

	// 3. Audit logs count
	var auditCount struct {
		Count int `alias:"count"`
	}
	auditStmt := SELECT(COUNT(table.AuditLogs.ID).AS("count")).
		FROM(table.AuditLogs).
		WHERE(table.AuditLogs.ProjectID.EQ(UUID(projectID)))
	if err := auditStmt.Query(h.db, &auditCount); err != nil {
		slog.Error("get project stats: audit logs count", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch project stats"})
		return
	}

	writeJSON(w, http.StatusOK, ProjectStatsResponse{
		TotalSecrets:       totalSecrets,
		SecretsByEnv:       secretsByEnv,
		TotalServiceTokens: tokensCount.Count,
		TotalAuditLogs:     auditCount.Count,
	})
}
