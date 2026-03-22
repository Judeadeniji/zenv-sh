package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// RecoveryHandler handles vault recovery endpoints.
type RecoveryHandler struct {
	db       *sql.DB
	identity *middleware.IdentitySession
}

func NewRecoveryHandler(db *sql.DB, identity *middleware.IdentitySession) *RecoveryHandler {
	return &RecoveryHandler{db: db, identity: identity}
}

// --- Recovery Status ---

type RecoveryStatusResponse struct {
	HasKit           bool   `json:"has_kit"`
	HasContact       bool   `json:"has_contact"`
	RecoveryDisabled bool   `json:"recovery_disabled"`
	ContactEmail     string `json:"contact_email,omitempty"`
}

// Status returns what recovery methods the user has configured.
//
//	@Summary		Recovery status
//	@Description	Returns which recovery methods are available for this user.
//	@Tags			recovery
//	@Produce		json
//	@Success		200	{object}	RecoveryStatusResponse
//	@Security		SessionAuth
//	@Router			/auth/recovery/status [get]
func (h *RecoveryHandler) Status(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var user model.Users
	stmt := SELECT(
		table.Users.ID,
		table.Users.RecoveryWrappedDek,
		table.Users.RecoveryDisabled,
	).FROM(table.Users).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))

	if err := stmt.Query(h.db, &user); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	hasKit := user.RecoveryWrappedDek != nil && len(*user.RecoveryWrappedDek) > 0

	// Check for trusted contact
	contactU := table.Users.AS("contact_user")
	var contactUser model.Users
	contactStmt := SELECT(
		contactU.Email,
	).FROM(
		table.TrustedContacts.INNER_JOIN(contactU, table.TrustedContacts.ContactUserID.EQ(contactU.ID)),
	).WHERE(table.TrustedContacts.UserID.EQ(UUID(user.ID)))

	hasContact := false
	contactEmail := ""
	if err := contactStmt.Query(h.db, &contactUser); err == nil {
		hasContact = true
		contactEmail = contactUser.Email
	}

	writeJSON(w, http.StatusOK, RecoveryStatusResponse{
		HasKit:           hasKit,
		HasContact:       hasContact,
		RecoveryDisabled: user.RecoveryDisabled,
		ContactEmail:     contactEmail,
	})
}

// --- Recovery Kit: Fetch ---

type RecoveryKitResponse struct {
	RecoveryWrappedDEK string `json:"recovery_wrapped_dek"` // base64
}

// GetKit returns the recovery_wrapped_dek for the authenticated user.
//
//	@Summary		Get recovery kit material
//	@Description	Returns the recovery-wrapped DEK so the client can attempt recovery.
//	@Tags			recovery
//	@Produce		json
//	@Success		200	{object}	RecoveryKitResponse
//	@Failure		403	{object}	ErrorResponse	"Recovery disabled"
//	@Failure		404	{object}	ErrorResponse	"No recovery kit"
//	@Security		SessionAuth
//	@Router			/auth/recovery/kit [get]
func (h *RecoveryHandler) GetKit(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var user model.Users
	stmt := SELECT(
		table.Users.RecoveryWrappedDek,
		table.Users.RecoveryDisabled,
	).FROM(table.Users).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))

	if err := stmt.Query(h.db, &user); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	if user.RecoveryDisabled {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "recovery is disabled for this account"})
		return
	}

	if user.RecoveryWrappedDek == nil || len(*user.RecoveryWrappedDek) == 0 {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "no recovery kit configured"})
		return
	}

	writeJSON(w, http.StatusOK, RecoveryKitResponse{
		RecoveryWrappedDEK: base64.StdEncoding.EncodeToString(*user.RecoveryWrappedDek),
	})
}

// --- Recovery Kit: Recover ---

type RecoverWithKitRequest struct {
	NewVaultKeyType       string `json:"new_vault_key_type"`
	NewSalt               string `json:"new_salt"`
	NewAuthKeyHash        string `json:"new_auth_key_hash"`
	NewWrappedDEK         string `json:"new_wrapped_dek"`
	NewWrappedPrivateKey  string `json:"new_wrapped_private_key"`
	NewRecoveryWrappedDEK string `json:"new_recovery_wrapped_dek"`
}

// RecoverWithKit replaces vault crypto material after successful client-side recovery.
//
//	@Summary		Recover with Recovery Kit
//	@Description	Client verified recovery words, unwrapped DEK, set new Vault Key. Submit new crypto material.
//	@Tags			recovery
//	@Accept			json
//	@Produce		json
//	@Param			body	body	RecoverWithKitRequest	true	"New crypto material"
//	@Success		200		{object}	map[string]string
//	@Failure		403		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/auth/recovery/kit/recover [post]
func (h *RecoveryHandler) RecoverWithKit(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var req RecoverWithKitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.NewVaultKeyType != "pin" && req.NewVaultKeyType != "passphrase" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "new_vault_key_type must be 'pin' or 'passphrase'"})
		return
	}

	newSalt, err := base64.StdEncoding.DecodeString(req.NewSalt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_salt"})
		return
	}
	newAuthKeyHash, err := base64.StdEncoding.DecodeString(req.NewAuthKeyHash)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_auth_key_hash"})
		return
	}
	newWrappedDEK, err := base64.StdEncoding.DecodeString(req.NewWrappedDEK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_wrapped_dek"})
		return
	}
	newWrappedPrivateKey, err := base64.StdEncoding.DecodeString(req.NewWrappedPrivateKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_wrapped_private_key"})
		return
	}

	var newRecoveryWrappedDEK []byte
	if req.NewRecoveryWrappedDEK != "" {
		newRecoveryWrappedDEK, err = base64.StdEncoding.DecodeString(req.NewRecoveryWrappedDEK)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_recovery_wrapped_dek"})
			return
		}
	}

	now := time.Now().UTC()
	updateStmt := table.Users.UPDATE(
		table.Users.VaultKeyType,
		table.Users.Salt,
		table.Users.AuthKeyHash,
		table.Users.WrappedDek,
		table.Users.WrappedPrivateKey,
		table.Users.RecoveryWrappedDek,
		table.Users.UpdatedAt,
	).SET(
		req.NewVaultKeyType,
		newSalt,
		newAuthKeyHash,
		newWrappedDEK,
		newWrappedPrivateKey,
		newRecoveryWrappedDEK,
		now,
	).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))

	if _, err := updateStmt.Exec(h.db); err != nil {
		slog.Error("recover-with-kit: update user", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update vault"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "vault recovered"})
}

// --- Trusted Contact: Set ---

type SetTrustedContactRequest struct {
	ContactEmail      string `json:"contact_email"`
	TrustedWrappedDEK string `json:"trusted_wrapped_dek"` // base64
}

// SetTrustedContact designates a trusted contact for recovery.
//
//	@Summary		Set trusted contact
//	@Description	Wrap DEK with contact's public key and store. Requires unlocked vault.
//	@Tags			recovery
//	@Accept			json
//	@Produce		json
//	@Param			body	body	SetTrustedContactRequest	true	"Contact email + wrapped DEK"
//	@Success		201		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/recovery/trusted-contact [post]
func (h *RecoveryHandler) SetTrustedContact(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var req SetTrustedContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	trustedWrappedDEK, err := base64.StdEncoding.DecodeString(req.TrustedWrappedDEK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in trusted_wrapped_dek"})
		return
	}

	// Get current user
	var user model.Users
	userStmt := SELECT(
		table.Users.ID,
		table.Users.RecoveryDisabled,
	).FROM(table.Users).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))

	if err := userStmt.Query(h.db, &user); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	if user.RecoveryDisabled {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "recovery is disabled for this account"})
		return
	}

	// Find contact user by email
	var contactUser model.Users
	contactStmt := SELECT(table.Users.ID).FROM(table.Users).WHERE(table.Users.Email.EQ(String(req.ContactEmail)))
	if err := contactStmt.Query(h.db, &contactUser); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "contact user not found"})
		return
	}

	if contactUser.ID == user.ID {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "cannot designate yourself as trusted contact"})
		return
	}

	// Upsert trusted contact
	insertStmt := table.TrustedContacts.INSERT(
		table.TrustedContacts.UserID,
		table.TrustedContacts.ContactUserID,
		table.TrustedContacts.TrustedWrappedDek,
	).VALUES(
		user.ID,
		contactUser.ID,
		trustedWrappedDEK,
	).ON_CONFLICT(table.TrustedContacts.UserID, table.TrustedContacts.ContactUserID).DO_UPDATE(
		SET(table.TrustedContacts.TrustedWrappedDek.SET(table.TrustedContacts.EXCLUDED.TrustedWrappedDek)),
	)

	if _, err := insertStmt.Exec(h.db); err != nil {
		slog.Error("set-trusted-contact: upsert", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to set trusted contact"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "trusted contact set"})
}

// --- Trusted Contact: Remove ---

// RemoveTrustedContact removes the user's trusted contact.
//
//	@Summary		Remove trusted contact
//	@Tags			recovery
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/recovery/trusted-contact [delete]
func (h *RecoveryHandler) RemoveTrustedContact(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	// Get user ID
	var user model.Users
	userStmt := SELECT(table.Users.ID).FROM(table.Users).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))
	if err := userStmt.Query(h.db, &user); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	deleteStmt := table.TrustedContacts.DELETE().WHERE(table.TrustedContacts.UserID.EQ(UUID(user.ID)))
	if _, err := deleteStmt.Exec(h.db); err != nil {
		slog.Error("remove-trusted-contact: delete", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to remove trusted contact"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "trusted contact removed"})
}

// --- Recovery Request: Initiate ---

type InitiateRecoveryRequest struct {
	RecoveryPublicKey string `json:"recovery_public_key"` // base64
}

// InitiateRecovery starts a 72-hour trusted contact recovery request.
//
//	@Summary		Initiate recovery request
//	@Description	Start 72-hour waiting period for trusted contact recovery.
//	@Tags			recovery
//	@Accept			json
//	@Produce		json
//	@Param			body	body	InitiateRecoveryRequest	true	"Ephemeral public key"
//	@Success		201		{object}	map[string]any
//	@Security		SessionAuth
//	@Router			/auth/recovery/request [post]
func (h *RecoveryHandler) InitiateRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var req InitiateRecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	recoveryPubKey, err := base64.StdEncoding.DecodeString(req.RecoveryPublicKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in recovery_public_key"})
		return
	}

	// Get user + trusted contact via join
	var tc model.TrustedContacts
	tcStmt := SELECT(
		table.TrustedContacts.UserID,
		table.TrustedContacts.ContactUserID,
	).FROM(
		table.TrustedContacts.INNER_JOIN(table.Users, table.TrustedContacts.UserID.EQ(table.Users.ID)),
	).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))

	if err := tcStmt.Query(h.db, &tc); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "no trusted contact configured"})
		return
	}

	now := time.Now().UTC()
	eligibleAt := now.Add(72 * time.Hour)
	requestID := uuid.New()

	insertStmt := table.RecoveryRequests.INSERT(
		table.RecoveryRequests.ID,
		table.RecoveryRequests.UserID,
		table.RecoveryRequests.ContactUserID,
		table.RecoveryRequests.RecoveryPublicKey,
		table.RecoveryRequests.RequestedAt,
		table.RecoveryRequests.EligibleAt,
	).VALUES(
		requestID,
		tc.UserID,
		tc.ContactUserID,
		recoveryPubKey,
		now,
		eligibleAt,
	)

	if _, err := insertStmt.Exec(h.db); err != nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "active recovery request already exists"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"request_id":  requestID.String(),
		"eligible_at": eligibleAt.Format(time.RFC3339),
		"status":      "pending",
	})
}

// --- Recovery Request: Cancel ---

// CancelRecovery cancels the user's active recovery request.
//
//	@Summary		Cancel recovery request
//	@Tags			recovery
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/recovery/request [delete]
func (h *RecoveryHandler) CancelRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var user model.Users
	userStmt := SELECT(table.Users.ID).FROM(table.Users).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))
	if err := userStmt.Query(h.db, &user); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	now := time.Now().UTC()
	updateStmt := table.RecoveryRequests.UPDATE(
		table.RecoveryRequests.Status,
		table.RecoveryRequests.CancelledAt,
	).SET(
		"cancelled",
		now,
	).WHERE(
		table.RecoveryRequests.UserID.EQ(UUID(user.ID)).
			AND(table.RecoveryRequests.Status.IN(String("pending"), String("approved"))),
	)

	if _, err := updateStmt.Exec(h.db); err != nil {
		slog.Error("cancel-recovery: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to cancel"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// --- Recovery Request: Status ---

type RecoveryRequestStatusResponse struct {
	RequestID   string `json:"request_id"`
	Status      string `json:"status"`
	EligibleAt  string `json:"eligible_at"`
	HasPayload  bool   `json:"has_payload"`
	RequestedAt string `json:"requested_at"`
}

// GetRecoveryRequest returns the status of the user's active recovery request.
//
//	@Summary		Recovery request status
//	@Tags			recovery
//	@Produce		json
//	@Success		200	{object}	RecoveryRequestStatusResponse
//	@Security		SessionAuth
//	@Router			/auth/recovery/request [get]
func (h *RecoveryHandler) GetRecoveryRequest(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var rr model.RecoveryRequests
	stmt := SELECT(
		table.RecoveryRequests.ID,
		table.RecoveryRequests.Status,
		table.RecoveryRequests.EligibleAt,
		table.RecoveryRequests.RequestedAt,
		table.RecoveryRequests.RecoveryPayload,
	).FROM(
		table.RecoveryRequests.INNER_JOIN(table.Users, table.RecoveryRequests.UserID.EQ(table.Users.ID)),
	).WHERE(
		table.Users.IdentityID.EQ(String(sess.IdentityID)).
			AND(table.RecoveryRequests.Status.IN(String("pending"), String("approved"))),
	).ORDER_BY(table.RecoveryRequests.RequestedAt.DESC()).LIMIT(1)

	if err := stmt.Query(h.db, &rr); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "no active recovery request"})
		return
	}

	writeJSON(w, http.StatusOK, RecoveryRequestStatusResponse{
		RequestID:   rr.ID.String(),
		Status:      rr.Status,
		EligibleAt:  rr.EligibleAt.Format(time.RFC3339),
		HasPayload:  rr.RecoveryPayload != nil && len(*rr.RecoveryPayload) > 0,
		RequestedAt: rr.RequestedAt.Format(time.RFC3339),
	})
}

// --- Incoming Requests (for trusted contacts) ---

type IncomingRequest struct {
	RequestID   string `json:"request_id"`
	UserEmail   string `json:"user_email"`
	Status      string `json:"status"`
	EligibleAt  string `json:"eligible_at"`
	RequestedAt string `json:"requested_at"`
}

// GetIncomingRequests lists recovery requests where the authenticated user is the trusted contact.
//
//	@Summary		Incoming recovery requests
//	@Tags			recovery
//	@Produce		json
//	@Success		200	{array}		IncomingRequest
//	@Security		SessionAuth
//	@Router			/auth/recovery/incoming-requests [get]
func (h *RecoveryHandler) GetIncomingRequests(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	// Get contact's user ID
	var contactUser model.Users
	contactStmt := SELECT(table.Users.ID).FROM(table.Users).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))
	if err := contactStmt.Query(h.db, &contactUser); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	// Fetch active requests where this user is the contact
	requestingUser := table.Users.AS("requesting_user")

	type result struct {
		model.RecoveryRequests
		RequestingUser model.Users `alias:"requesting_user"`
	}

	var results []result
	stmt := SELECT(
		table.RecoveryRequests.ID,
		table.RecoveryRequests.Status,
		table.RecoveryRequests.EligibleAt,
		table.RecoveryRequests.RequestedAt,
		requestingUser.Email,
	).FROM(
		table.RecoveryRequests.INNER_JOIN(requestingUser, table.RecoveryRequests.UserID.EQ(requestingUser.ID)),
	).WHERE(
		table.RecoveryRequests.ContactUserID.EQ(UUID(contactUser.ID)).
			AND(table.RecoveryRequests.Status.IN(String("pending"), String("approved"))),
	).ORDER_BY(table.RecoveryRequests.RequestedAt.DESC())

	if err := stmt.Query(h.db, &results); err != nil && !errors.Is(err, qrm.ErrNoRows) {
		slog.Error("incoming-requests: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch requests"})
		return
	}

	requests := make([]IncomingRequest, 0, len(results))
	for _, res := range results {
		requests = append(requests, IncomingRequest{
			RequestID:   res.ID.String(),
			UserEmail:   res.RequestingUser.Email,
			Status:      res.Status,
			EligibleAt:  res.EligibleAt.Format(time.RFC3339),
			RequestedAt: res.RequestedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, requests)
}

// --- Approve Recovery Request ---

type ApproveRecoveryRequest struct {
	RecoveryPayload string `json:"recovery_payload"` // base64
}

// ApproveRecovery lets a trusted contact approve a recovery request after 72h.
//
//	@Summary		Approve recovery request
//	@Description	Trusted contact provides DEK re-wrapped with recovering user's ephemeral key.
//	@Tags			recovery
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string					true	"Recovery request ID"
//	@Param			body	body	ApproveRecoveryRequest	true	"Recovery payload"
//	@Success		200		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/recovery/request/{id}/approve [post]
func (h *RecoveryHandler) ApproveRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	requestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request ID"})
		return
	}

	var req ApproveRecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	payload, err := base64.StdEncoding.DecodeString(req.RecoveryPayload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in recovery_payload"})
		return
	}

	// Fetch the request
	var rr model.RecoveryRequests
	rrStmt := SELECT(
		table.RecoveryRequests.ContactUserID,
		table.RecoveryRequests.EligibleAt,
		table.RecoveryRequests.Status,
	).FROM(table.RecoveryRequests).WHERE(table.RecoveryRequests.ID.EQ(UUID(requestID)))

	if err := rrStmt.Query(h.db, &rr); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "recovery request not found"})
		return
	}

	// Verify the authenticated user is the trusted contact
	var contactUser model.Users
	contactStmt := SELECT(table.Users.IdentityID).FROM(table.Users).WHERE(table.Users.ID.EQ(UUID(rr.ContactUserID)))
	if err := contactStmt.Query(h.db, &contactUser); err != nil || contactUser.IdentityID == nil || *contactUser.IdentityID != sess.IdentityID {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "you are not the trusted contact for this request"})
		return
	}

	if rr.Status != "pending" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "request is not pending"})
		return
	}

	if time.Now().Before(rr.EligibleAt) {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "72-hour waiting period has not elapsed"})
		return
	}

	now := time.Now().UTC()
	updateStmt := table.RecoveryRequests.UPDATE(
		table.RecoveryRequests.Status,
		table.RecoveryRequests.RecoveryPayload,
		table.RecoveryRequests.ApprovedAt,
	).SET(
		"approved",
		payload,
		now,
	).WHERE(table.RecoveryRequests.ID.EQ(UUID(requestID)))

	if _, err := updateStmt.Exec(h.db); err != nil {
		slog.Error("approve-recovery: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to approve"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// --- Complete Recovery ---

type CompleteRecoveryRequest struct {
	RequestID             string `json:"request_id"`
	NewVaultKeyType       string `json:"new_vault_key_type"`
	NewSalt               string `json:"new_salt"`
	NewAuthKeyHash        string `json:"new_auth_key_hash"`
	NewWrappedDEK         string `json:"new_wrapped_dek"`
	NewWrappedPrivateKey  string `json:"new_wrapped_private_key"`
	NewRecoveryWrappedDEK string `json:"new_recovery_wrapped_dek"`
}

// CompleteRecovery finalizes trusted contact recovery by replacing vault crypto material.
//
//	@Summary		Complete trusted contact recovery
//	@Tags			recovery
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string						true	"Recovery request ID"
//	@Param			body	body	CompleteRecoveryRequest		true	"New crypto material"
//	@Success		200		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/recovery/request/{id}/complete [post]
func (h *RecoveryHandler) CompleteRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	requestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request ID"})
		return
	}

	var req CompleteRecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	// Verify this request belongs to the authenticated user and is approved
	var rr model.RecoveryRequests
	rrStmt := SELECT(
		table.RecoveryRequests.UserID,
		table.RecoveryRequests.Status,
	).FROM(
		table.RecoveryRequests.INNER_JOIN(table.Users, table.RecoveryRequests.UserID.EQ(table.Users.ID)),
	).WHERE(
		table.RecoveryRequests.ID.EQ(UUID(requestID)).
			AND(table.Users.IdentityID.EQ(String(sess.IdentityID))),
	)

	if err := rrStmt.Query(h.db, &rr); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "recovery request not found"})
		return
	}
	if rr.Status != "approved" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "request is not approved"})
		return
	}

	// Decode new crypto material
	newSalt, _ := base64.StdEncoding.DecodeString(req.NewSalt)
	newAuthKeyHash, _ := base64.StdEncoding.DecodeString(req.NewAuthKeyHash)
	newWrappedDEK, _ := base64.StdEncoding.DecodeString(req.NewWrappedDEK)
	newWrappedPrivateKey, _ := base64.StdEncoding.DecodeString(req.NewWrappedPrivateKey)
	var newRecoveryWrappedDEK []byte
	if req.NewRecoveryWrappedDEK != "" {
		newRecoveryWrappedDEK, _ = base64.StdEncoding.DecodeString(req.NewRecoveryWrappedDEK)
	}

	now := time.Now().UTC()

	// Update user crypto material
	userUpdate := table.Users.UPDATE(
		table.Users.VaultKeyType,
		table.Users.Salt,
		table.Users.AuthKeyHash,
		table.Users.WrappedDek,
		table.Users.WrappedPrivateKey,
		table.Users.RecoveryWrappedDek,
		table.Users.UpdatedAt,
	).SET(
		req.NewVaultKeyType,
		newSalt,
		newAuthKeyHash,
		newWrappedDEK,
		newWrappedPrivateKey,
		newRecoveryWrappedDEK,
		now,
	).WHERE(table.Users.ID.EQ(UUID(rr.UserID)))

	if _, err := userUpdate.Exec(h.db); err != nil {
		slog.Error("complete-recovery: update user", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update vault"})
		return
	}

	// Mark request as completed
	rrUpdate := table.RecoveryRequests.UPDATE(
		table.RecoveryRequests.Status,
		table.RecoveryRequests.CompletedAt,
	).SET(
		"completed",
		now,
	).WHERE(table.RecoveryRequests.ID.EQ(UUID(requestID)))

	if _, err := rrUpdate.Exec(h.db); err != nil {
		slog.Error("complete-recovery: update request", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to complete request"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "vault recovered"})
}

// --- Public Key Lookup ---

type PublicKeyResponse struct {
	Email     string `json:"email"`
	PublicKey string `json:"public_key"` // base64
}

// GetPublicKey looks up a user's public key by email.
//
//	@Summary		Lookup user public key
//	@Tags			recovery
//	@Produce		json
//	@Param			email	query	string	true	"User email"
//	@Success		200		{object}	PublicKeyResponse
//	@Security		SessionAuth
//	@Router			/users/public-key [get]
func (h *RecoveryHandler) GetPublicKey(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "email query param required"})
		return
	}

	var user model.Users
	stmt := SELECT(table.Users.PublicKey).FROM(table.Users).WHERE(table.Users.Email.EQ(String(email)))
	if err := stmt.Query(h.db, &user); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, PublicKeyResponse{
		Email:     email,
		PublicKey: base64.StdEncoding.EncodeToString(user.PublicKey),
	})
}
