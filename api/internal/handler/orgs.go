package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
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

// OrgsHandler handles organization CRUD and member management.
type OrgsHandler struct {
	db *sql.DB
}

func NewOrgsHandler(db *sql.DB) *OrgsHandler {
	return &OrgsHandler{db: db}
}

// --- Create Organization ---

type CreateOrgRequest struct {
	Name string `json:"name"`
}

type OrgResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OwnerID   string `json:"owner_id"`
	CreatedAt string `json:"created_at"`
}

// @Summary		Create organization
// @Description	Create an organization. The creating user becomes the owner and is added as an admin member.
// @Tags			organizations
// @Accept			json
// @Produce		json
// @Param			body	body		CreateOrgRequest	true	"Organization name"
// @Success		201		{object}	OrgResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/orgs [post]
func (h *OrgsHandler) Create(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "name is required"})
		return
	}

	userID, err := uuid.Parse(sess.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "invalid session"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		slog.Error("orgs.create: begin tx", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}
	defer tx.Rollback()

	orgID := uuid.New()
	now := time.Now().UTC()

	// 1. Insert organization
	insertOrg := table.Organizations.INSERT(
		table.Organizations.ID,
		table.Organizations.Name,
		table.Organizations.OwnerID,
		table.Organizations.CreatedAt,
	).VALUES(orgID, req.Name, userID, now)

	if _, err := insertOrg.Exec(tx); err != nil {
		slog.Error("orgs.create: insert org", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create organization"})
		return
	}

	// 2. Add creator as admin member
	insertMember := table.OrganizationMembers.INSERT(
		table.OrganizationMembers.ID,
		table.OrganizationMembers.OrganizationID,
		table.OrganizationMembers.UserID,
		table.OrganizationMembers.Role,
		table.OrganizationMembers.JoinedAt,
	).VALUES(uuid.New(), orgID, userID, "admin", now)

	if _, err := insertMember.Exec(tx); err != nil {
		slog.Error("orgs.create: insert member", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to add owner as member"})
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("orgs.create: commit", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to commit"})
		return
	}

	writeJSON(w, http.StatusCreated, OrgResponse{
		ID:        orgID.String(),
		Name:      req.Name,
		OwnerID:   userID.String(),
		CreatedAt: now.Format(time.RFC3339),
	})
}

// --- List Organizations ---

type ListOrgsResponse struct {
	Organizations []OrgResponse `json:"organizations"`
}

// @Summary		List organizations
// @Description	List all organizations the current user is a member of.
// @Tags			organizations
// @Produce		json
// @Success		200	{object}	ListOrgsResponse
// @Security		SessionAuth
// @Router			/orgs [get]
func (h *OrgsHandler) List(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	userID, err := uuid.Parse(sess.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "invalid session"})
		return
	}

	var orgs []model.Organizations
	stmt := SELECT(
		table.Organizations.ID,
		table.Organizations.Name,
		table.Organizations.OwnerID,
		table.Organizations.CreatedAt,
	).FROM(
		table.Organizations.INNER_JOIN(
			table.OrganizationMembers,
			table.OrganizationMembers.OrganizationID.EQ(table.Organizations.ID),
		),
	).WHERE(
		table.OrganizationMembers.UserID.EQ(UUID(userID)),
	).ORDER_BY(table.Organizations.Name.ASC())

	if err := stmt.Query(h.db, &orgs); err != nil && !errors.Is(err, qrm.ErrNoRows) {
		slog.Error("orgs.list: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list organizations"})
		return
	}

	resp := ListOrgsResponse{Organizations: make([]OrgResponse, 0, len(orgs))}
	for _, o := range orgs {
		resp.Organizations = append(resp.Organizations, OrgResponse{
			ID:        o.ID.String(),
			Name:      o.Name,
			OwnerID:   o.OwnerID.String(),
			CreatedAt: o.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Get Organization ---

// @Summary		Get organization
// @Description	Get a single organization by ID.
// @Tags			organizations
// @Produce		json
// @Param			orgID	path		string	true	"Organization UUID"
// @Success		200		{object}	OrgResponse
// @Failure		404		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/orgs/{orgID} [get]
func (h *OrgsHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid organization ID"})
		return
	}

	var org model.Organizations
	stmt := SELECT(
		table.Organizations.ID,
		table.Organizations.Name,
		table.Organizations.OwnerID,
		table.Organizations.CreatedAt,
	).FROM(table.Organizations).WHERE(table.Organizations.ID.EQ(UUID(orgID)))

	if err := stmt.Query(h.db, &org); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "organization not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get organization"})
		return
	}

	writeJSON(w, http.StatusOK, OrgResponse{
		ID:        org.ID.String(),
		Name:      org.Name,
		OwnerID:   org.OwnerID.String(),
		CreatedAt: org.CreatedAt.Format(time.RFC3339),
	})
}

// --- List Members ---

type MemberResponse struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

type ListMembersResponse struct {
	Members []MemberResponse `json:"members"`
}

// memberRow is used to scan the JOIN result.
type memberRow struct {
	ID       uuid.UUID `sql:"primary_key" alias:"organization_members.id"`
	UserID   uuid.UUID `alias:"organization_members.user_id"`
	Role     string    `alias:"organization_members.role"`
	JoinedAt time.Time `alias:"organization_members.joined_at"`
	Email    string    `alias:"users.email"`
}

// @Summary		List organization members
// @Description	List all members of an organization with their roles.
// @Tags			organizations
// @Produce		json
// @Param			orgID	path		string	true	"Organization UUID"
// @Success		200		{object}	ListMembersResponse
// @Failure		400		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/orgs/{orgID}/members [get]
func (h *OrgsHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid organization ID"})
		return
	}

	var rows []memberRow
	stmt := SELECT(
		table.OrganizationMembers.ID,
		table.OrganizationMembers.UserID,
		table.OrganizationMembers.Role,
		table.OrganizationMembers.JoinedAt,
		table.Users.Email,
	).FROM(
		table.OrganizationMembers.INNER_JOIN(
			table.Users,
			table.Users.ID.EQ(table.OrganizationMembers.UserID),
		),
	).WHERE(
		table.OrganizationMembers.OrganizationID.EQ(UUID(orgID)),
	).ORDER_BY(table.OrganizationMembers.JoinedAt.ASC())

	if err := stmt.Query(h.db, &rows); err != nil && !errors.Is(err, qrm.ErrNoRows) {
		slog.Error("orgs.list_members: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list members"})
		return
	}

	resp := ListMembersResponse{Members: make([]MemberResponse, 0, len(rows))}
	for _, m := range rows {
		resp.Members = append(resp.Members, MemberResponse{
			ID:       m.ID.String(),
			UserID:   m.UserID.String(),
			Email:    m.Email,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Add Member ---

type AddMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"` // admin, senior_dev, dev, contractor, ci_bot
}

// @Summary		Add organization member
// @Description	Add a user to an organization with a specified role.
// @Tags			organizations
// @Accept			json
// @Produce		json
// @Param			orgID	path		string			true	"Organization UUID"
// @Param			body	body		AddMemberRequest	true	"User and role"
// @Success		201		{object}	MemberResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/orgs/{orgID}/members [post]
func (h *OrgsHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid organization ID"})
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.UserID == "" || req.Role == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "user_id and role are required"})
		return
	}

	validRoles := map[string]bool{"admin": true, "senior_dev": true, "dev": true, "contractor": true, "ci_bot": true}
	if !validRoles[req.Role] {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "role must be one of: admin, senior_dev, dev, contractor, ci_bot"})
		return
	}

	// Verify caller is an admin of this org.
	callerID, _ := uuid.Parse(sess.UserID)
	if !h.isOrgAdmin(callerID, orgID) {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "only organization admins can add members"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user_id"})
		return
	}

	memberID := uuid.New()
	now := time.Now().UTC()

	insertStmt := table.OrganizationMembers.INSERT(
		table.OrganizationMembers.ID,
		table.OrganizationMembers.OrganizationID,
		table.OrganizationMembers.UserID,
		table.OrganizationMembers.Role,
		table.OrganizationMembers.JoinedAt,
	).VALUES(memberID, orgID, userID, req.Role, now)

	if _, err := insertStmt.Exec(h.db); err != nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "user is already a member of this organization"})
		return
	}

	writeJSON(w, http.StatusCreated, MemberResponse{
		ID:       memberID.String(),
		UserID:   userID.String(),
		Role:     req.Role,
		JoinedAt: now.Format(time.RFC3339),
	})
}

// --- Remove Member ---

// @Summary		Remove organization member
// @Description	Remove a member from an organization.
// @Tags			organizations
// @Produce		json
// @Param			orgID		path	string	true	"Organization UUID"
// @Param			memberID	path	string	true	"Member UUID"
// @Success		200			{object}	map[string]string
// @Failure		403			{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/orgs/{orgID}/members/{memberID} [delete]
func (h *OrgsHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid organization ID"})
		return
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "memberID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid member ID"})
		return
	}

	callerID, _ := uuid.Parse(sess.UserID)
	if !h.isOrgAdmin(callerID, orgID) {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "only organization admins can remove members"})
		return
	}

	// Prevent removing the org owner.
	var org model.Organizations
	ownerStmt := SELECT(table.Organizations.OwnerID).FROM(table.Organizations).WHERE(
		table.Organizations.ID.EQ(UUID(orgID)),
	)
	if err := ownerStmt.Query(h.db, &org); err == nil {
		var member model.OrganizationMembers
		memberStmt := SELECT(table.OrganizationMembers.UserID).FROM(table.OrganizationMembers).WHERE(
			table.OrganizationMembers.ID.EQ(UUID(memberID)),
		)
		if err := memberStmt.Query(h.db, &member); err == nil {
			if member.UserID == org.OwnerID {
				writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "cannot remove the organization owner"})
				return
			}
		}
	}

	deleteStmt := table.OrganizationMembers.DELETE().WHERE(
		table.OrganizationMembers.ID.EQ(UUID(memberID)).
			AND(table.OrganizationMembers.OrganizationID.EQ(UUID(orgID))),
	)

	result, err := deleteStmt.Exec(h.db)
	if err != nil {
		slog.Error("orgs.remove_member: delete", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to remove member"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "member not found in this organization"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "member removed"})
}

// isOrgAdmin checks if the user is an admin member of the organization.
func (h *OrgsHandler) isOrgAdmin(userID, orgID uuid.UUID) bool {
	var member model.OrganizationMembers
	stmt := SELECT(table.OrganizationMembers.Role).FROM(table.OrganizationMembers).WHERE(
		table.OrganizationMembers.OrganizationID.EQ(UUID(orgID)).
			AND(table.OrganizationMembers.UserID.EQ(UUID(userID))),
	)
	if err := stmt.Query(h.db, &member); err != nil {
		return false
	}
	return member.Role == "admin"
}
