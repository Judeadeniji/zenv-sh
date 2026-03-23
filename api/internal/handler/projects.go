package handler

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"encoding/json"
	"log/slog"
	"net/http"
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
}

// @Summary		List projects
// @Description	List all projects in an organization.
// @Tags			projects
// @Produce		json
// @Param			organization_id	query		string	true	"Organization ID"
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

	var projects []model.Projects
	stmt := SELECT(
		table.Projects.ID,
		table.Projects.OrganizationID,
		table.Projects.Name,
		table.Projects.CreatedAt,
	).FROM(table.Projects).WHERE(
		table.Projects.OrganizationID.EQ(UUID(orgID)),
	).ORDER_BY(table.Projects.Name.ASC())

	err = stmt.Query(h.db, &projects)
	if err != nil && !errors.Is(err, qrm.ErrNoRows) {
		slog.Error("projects.list: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list projects"})
		return
	}

	resp := ListProjectsResponse{Projects: make([]ProjectResponse, 0, len(projects))}
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
	ProjectSalt       string `json:"project_salt"`                  // base64
	WrappedProjectDEK string `json:"wrapped_project_dek"`           // base64
	VaultKeyType      string `json:"vault_key_type,omitempty"`      // "pin" or "passphrase"
}

// @Summary		Get project crypto
// @Description	Returns project salt and wrapped Project DEK. SDK uses this to derive Project KEK and unwrap DEK.
// @Tags			projects
// @Produce		json
// @Param			projectID	path		string	true	"Project UUID"
// @Success		200			{object}	ProjectCryptoResponse
// @Failure		404			{object}	ErrorResponse
// @Security		BearerAuth
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
