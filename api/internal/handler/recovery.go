package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/Judeadeniji/zenv-sh/api/internal/user_lookup"
)

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

// @Summary		Recovery status
// @Description	Returns which recovery methods are available for this user.
// @Tags			recovery
// @Produce		json
// @Success		200	{object}	RecoveryStatusResponse
// @Failure		401	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/status [get]
func (h *RecoveryHandler) Status(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var identity model.Identities
	if err := SELECT(
		table.Identities.ID,
		table.Identities.RecoveryWrappedDek,
		table.Identities.RecoveryDisabled,
	).FROM(table.Identities).WHERE(table.Identities.IdentityID.EQ(UUID(userID))).
		Query(h.db, &identity); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	hasKit := identity.RecoveryWrappedDek != nil && len(*identity.RecoveryWrappedDek) > 0

	hasContact := false
	contactEmail := ""
	var contactRow struct {
		ContactUserID uuid.UUID `alias:"trusted_contacts.contact_user_id"`
	}
	if err := SELECT(table.TrustedContacts.ContactUserID).
		FROM(table.TrustedContacts).
		WHERE(table.TrustedContacts.UserID.EQ(UUID(userID))).
		LIMIT(1).
		QueryContext(r.Context(), h.db, &contactRow); err != nil {
		if !errors.Is(err, qrm.ErrNoRows) {
			slog.Error("recovery status: contact lookup", "error", err)
		}
	} else {
		hasContact = true
		if _, em, ok, err := user_lookup.ByID(r.Context(), h.db, contactRow.ContactUserID); err == nil && ok && em != "" {
			contactEmail = em
		}
	}

	writeJSON(w, http.StatusOK, RecoveryStatusResponse{
		HasKit:           hasKit,
		HasContact:       hasContact,
		RecoveryDisabled: identity.RecoveryDisabled,
		ContactEmail:     contactEmail,
	})
}

// --- Disable / Enable Recovery ---

type DisableRecoveryRequest struct {
	Disabled bool `json:"disabled"`
}

// @Summary		Toggle recovery disabled
// @Tags			recovery
// @Accept			json
// @Produce		json
// @Param			body	body		DisableRecoveryRequest	true	"Toggle recovery"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/disable [put]
func (h *RecoveryHandler) DisableRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var req DisableRecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if _, err := table.Identities.UPDATE(table.Identities.RecoveryDisabled).
		SET(req.Disabled).
		WHERE(table.Identities.IdentityID.EQ(UUID(userID))).
		Exec(h.db); err != nil {
		slog.Error("disable recovery: update failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update recovery setting"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Recovery Kit: Fetch ---

type RecoveryKitResponse struct {
	RecoveryWrappedDEK string `json:"recovery_wrapped_dek"`
}

// @Summary		Get recovery kit material
// @Tags			recovery
// @Produce		json
// @Success		200	{object}	RecoveryKitResponse
// @Failure		401	{object}	ErrorResponse
// @Failure		403	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/kit [get]
func (h *RecoveryHandler) GetKit(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var identity model.Identities
	if err := SELECT(
		table.Identities.RecoveryWrappedDek,
		table.Identities.RecoveryDisabled,
	).FROM(table.Identities).WHERE(table.Identities.IdentityID.EQ(UUID(userID))).
		Query(h.db, &identity); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	if identity.RecoveryDisabled {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "recovery is disabled for this account"})
		return
	}
	if identity.RecoveryWrappedDek == nil || len(*identity.RecoveryWrappedDek) == 0 {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "no recovery kit configured"})
		return
	}

	writeJSON(w, http.StatusOK, RecoveryKitResponse{
		RecoveryWrappedDEK: base64.StdEncoding.EncodeToString(*identity.RecoveryWrappedDek),
	})
}

// --- Recovery Kit: Regenerate ---

type RegenerateKitRequest struct {
	RecoveryWrappedDEK string `json:"recovery_wrapped_dek"`
}

// @Summary		Regenerate recovery kit
// @Tags			recovery
// @Accept			json
// @Produce		json
// @Param			body	body		RegenerateKitRequest	true	"New recovery wrapped DEK"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/kit [put]
func (h *RecoveryHandler) RegenerateKit(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var req RegenerateKitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RecoveryWrappedDEK == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "recovery_wrapped_dek is required"})
		return
	}
	blob, err := base64.StdEncoding.DecodeString(req.RecoveryWrappedDEK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64"})
		return
	}

	var identity model.Identities
	if err := SELECT(table.Identities.ID, table.Identities.RecoveryDisabled).
		FROM(table.Identities).WHERE(table.Identities.IdentityID.EQ(UUID(userID))).
		Query(h.db, &identity); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	if identity.RecoveryDisabled {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "recovery is disabled for this account"})
		return
	}

	if _, err := table.Identities.UPDATE(table.Identities.RecoveryWrappedDek).
		SET(blob).
		WHERE(table.Identities.ID.EQ(UUID(identity.ID))).
		Exec(h.db); err != nil {
		slog.Error("regenerate kit: update failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update recovery kit"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

// @Summary		Recover with Recovery Kit
// @Tags			recovery
// @Accept			json
// @Produce		json
// @Param			body	body		RecoverWithKitRequest	true	"New crypto material"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/kit/recover [post]
func (h *RecoveryHandler) RecoverWithKit(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

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

	if _, err := table.Identities.UPDATE(
		table.Identities.VaultKeyType,
		table.Identities.Salt,
		table.Identities.AuthKeyHash,
		table.Identities.WrappedDek,
		table.Identities.WrappedPrivateKey,
		table.Identities.RecoveryWrappedDek,
		table.Identities.UpdatedAt,
	).SET(
		req.NewVaultKeyType,
		newSalt,
		newAuthKeyHash,
		newWrappedDEK,
		newWrappedPrivateKey,
		newRecoveryWrappedDEK,
		time.Now().UTC(),
	).WHERE(table.Identities.IdentityID.EQ(UUID(userID))).Exec(h.db); err != nil {
		slog.Error("recover-with-kit: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update vault"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "vault recovered"})
}

// --- Trusted Contact: Set ---

type SetTrustedContactRequest struct {
	ContactEmail      string `json:"contact_email"`
	TrustedWrappedDEK string `json:"trusted_wrapped_dek"`
}

// @Summary		Set trusted contact
// @Tags			recovery
// @Accept			json
// @Produce		json
// @Param			body	body		SetTrustedContactRequest	true	"Contact email + wrapped DEK"
// @Success		201		{object}	map[string]string
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/trusted-contact [post]
func (h *RecoveryHandler) SetTrustedContact(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

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

	var identity model.Identities
	if err := SELECT(table.Identities.ID, table.Identities.RecoveryDisabled).
		FROM(table.Identities).WHERE(table.Identities.IdentityID.EQ(UUID(userID))).
		Query(h.db, &identity); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	if identity.RecoveryDisabled {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "recovery is disabled for this account"})
		return
	}

	contactUserID, err := user_lookup.IDByEmail(r.Context(), h.db, req.ContactEmail)
	if err != nil {
		if errors.Is(err, user_lookup.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "contact user not found"})
			return
		}
		slog.Error("set-trusted-contact: lookup contact", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to resolve contact"})
		return
	}
	if contactUserID == userID {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "cannot designate yourself as trusted contact"})
		return
	}

	if _, err := table.TrustedContacts.INSERT(
		table.TrustedContacts.UserID,
		table.TrustedContacts.ContactUserID,
		table.TrustedContacts.TrustedWrappedDek,
	).VALUES(userID, contactUserID, trustedWrappedDEK).
		ON_CONFLICT(table.TrustedContacts.UserID, table.TrustedContacts.ContactUserID).
		DO_UPDATE(SET(table.TrustedContacts.TrustedWrappedDek.SET(table.TrustedContacts.EXCLUDED.TrustedWrappedDek))).
		Exec(h.db); err != nil {
		slog.Error("set-trusted-contact: upsert", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to set trusted contact"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "trusted contact set"})
}

// --- Trusted Contact: Remove ---

// @Summary		Remove trusted contact
// @Tags			recovery
// @Produce		json
// @Success		200	{object}	map[string]string
// @Failure		401	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/trusted-contact [delete]
func (h *RecoveryHandler) RemoveTrustedContact(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	if _, err := table.TrustedContacts.DELETE().
		WHERE(table.TrustedContacts.UserID.EQ(UUID(userID))).
		Exec(h.db); err != nil {
		slog.Error("remove-trusted-contact: delete", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to remove trusted contact"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "trusted contact removed"})
}

// --- Recovery Request: Initiate ---

type InitiateRecoveryRequest struct {
	RecoveryPublicKey string `json:"recovery_public_key"`
}

// @Summary		Initiate recovery request
// @Tags			recovery
// @Accept			json
// @Produce		json
// @Param			body	body		InitiateRecoveryRequest	true	"Ephemeral public key"
// @Success		201		{object}	map[string]any
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/request [post]
func (h *RecoveryHandler) InitiateRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

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

	var tc model.TrustedContacts
	if err := SELECT(table.TrustedContacts.UserID, table.TrustedContacts.ContactUserID).
		FROM(table.TrustedContacts).
		WHERE(table.TrustedContacts.UserID.EQ(UUID(userID))).
		Query(h.db, &tc); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "no trusted contact configured"})
		return
	}

	now := time.Now().UTC()
	eligibleAt := now.Add(72 * time.Hour)
	requestID := uuid.New()

	if _, err := table.RecoveryRequests.INSERT(
		table.RecoveryRequests.ID,
		table.RecoveryRequests.UserID,
		table.RecoveryRequests.ContactUserID,
		table.RecoveryRequests.RecoveryPublicKey,
		table.RecoveryRequests.RequestedAt,
		table.RecoveryRequests.EligibleAt,
	).VALUES(requestID, tc.UserID, tc.ContactUserID, recoveryPubKey, now, eligibleAt).
		Exec(h.db); err != nil {
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

// @Summary		Cancel recovery request
// @Tags			recovery
// @Produce		json
// @Success		200	{object}	map[string]string
// @Failure		401	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/request [delete]
func (h *RecoveryHandler) CancelRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	if _, err := table.RecoveryRequests.UPDATE(
		table.RecoveryRequests.Status,
		table.RecoveryRequests.CancelledAt,
	).SET("cancelled", time.Now().UTC()).WHERE(
		table.RecoveryRequests.UserID.EQ(UUID(userID)).
			AND(table.RecoveryRequests.Status.IN(NewEnumValue("pending"), NewEnumValue("approved"))),
	).Exec(h.db); err != nil {
		slog.Error("cancel-recovery: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to cancel"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// --- Recovery Request: Status ---

type RecoveryRequestStatusResponse struct {
	RequestID   string               `json:"request_id"`
	Status      model.RecoveryStatus `json:"status"`
	EligibleAt  string               `json:"eligible_at"`
	HasPayload  bool                 `json:"has_payload"`
	RequestedAt string               `json:"requested_at"`
}

// @Summary		Recovery request status
// @Tags			recovery
// @Produce		json
// @Success		200	{object}	RecoveryRequestStatusResponse
// @Failure		401	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/request [get]
func (h *RecoveryHandler) GetRecoveryRequest(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var rr model.RecoveryRequests
	if err := SELECT(
		table.RecoveryRequests.ID,
		table.RecoveryRequests.Status,
		table.RecoveryRequests.EligibleAt,
		table.RecoveryRequests.RequestedAt,
		table.RecoveryRequests.RecoveryPayload,
	).FROM(table.RecoveryRequests).WHERE(
		table.RecoveryRequests.UserID.EQ(UUID(userID)).
			AND(table.RecoveryRequests.Status.IN(NewEnumValue("pending"), NewEnumValue("approved"))),
	).ORDER_BY(table.RecoveryRequests.RequestedAt.DESC()).LIMIT(1).
		Query(h.db, &rr); err != nil {
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

// --- Incoming Requests ---

type IncomingRequest struct {
	RequestID      string               `json:"request_id"`
	RequesterName  string               `json:"requester_name"`
	RequesterEmail string               `json:"requester_email"`
	Status         model.RecoveryStatus `json:"status"`
	EligibleAt     string               `json:"eligible_at"`
	RequestedAt    string               `json:"requested_at"`
}

// @Summary		Incoming recovery requests
// @Tags			recovery
// @Produce		json
// @Success		200	{array}		IncomingRequest
// @Failure		401	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/incoming-requests [get]
func (h *RecoveryHandler) GetIncomingRequests(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	type result struct {
		model.RecoveryRequests
	}

	var results []result
	if err := SELECT(
		table.RecoveryRequests.ID,
		table.RecoveryRequests.UserID,
		table.RecoveryRequests.Status,
		table.RecoveryRequests.EligibleAt,
		table.RecoveryRequests.RequestedAt,
	).FROM(
		table.RecoveryRequests,
	).WHERE(
		table.RecoveryRequests.ContactUserID.EQ(UUID(userID)).
			AND(table.RecoveryRequests.Status.IN(NewEnumValue("pending"), NewEnumValue("approved"))),
	).ORDER_BY(table.RecoveryRequests.RequestedAt.DESC()).
		Query(h.db, &results); err != nil && !errors.Is(err, qrm.ErrNoRows) {
		slog.Error("incoming-requests: query", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch requests"})
		return
	}

	requests := make([]IncomingRequest, 0, len(results))
	for _, res := range results {
		reqName, reqEmail := "", ""
		if n, em, ok, err := user_lookup.ByID(r.Context(), h.db, res.UserID); err == nil && ok {
			reqName, reqEmail = n, em
		}
		requests = append(requests, IncomingRequest{
			RequestID:      res.ID.String(),
			RequesterName:  reqName,
			RequesterEmail: reqEmail,
			Status:         res.Status,
			EligibleAt:     res.EligibleAt.Format(time.RFC3339),
			RequestedAt:    res.RequestedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, requests)
}

// --- Approve Recovery Request ---

type ApproveRecoveryRequest struct {
	RecoveryPayload string `json:"recovery_payload"`
}

// @Summary		Approve recovery request
// @Tags			recovery
// @Accept			json
// @Produce		json
// @Param			id		path	string					true	"Recovery request ID"
// @Param			body	body	ApproveRecoveryRequest	true	"Recovery payload"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/request/{id}/approve [post]
func (h *RecoveryHandler) ApproveRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	callerID, _ := uuid.Parse(sess.UserID)

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

	var rr model.RecoveryRequests
	if err := SELECT(
		table.RecoveryRequests.ContactUserID,
		table.RecoveryRequests.EligibleAt,
		table.RecoveryRequests.Status,
	).FROM(table.RecoveryRequests).WHERE(table.RecoveryRequests.ID.EQ(UUID(requestID))).
		Query(h.db, &rr); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "recovery request not found"})
		return
	}

	// contact_user_id references users.id directly — compare with caller's users.id
	if rr.ContactUserID != callerID {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "you are not the trusted contact for this request"})
		return
	}
	if rr.Status != "pending" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "request is not pending"})
		return
	}
	if time.Now().Before(rr.EligibleAt) {
		days := int(time.Until(rr.EligibleAt).Hours() / 24)
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: fmt.Sprintf("72-hour waiting period has not elapsed, %d days left", days)})
		return
	}

	if _, err := table.RecoveryRequests.UPDATE(
		table.RecoveryRequests.Status,
		table.RecoveryRequests.RecoveryPayload,
		table.RecoveryRequests.ApprovedAt,
	).SET("approved", payload, time.Now().UTC()).
		WHERE(table.RecoveryRequests.ID.EQ(UUID(requestID))).
		Exec(h.db); err != nil {
		slog.Error("approve-recovery: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to approve"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// --- Complete Recovery ---

type CompleteRecoveryRequest struct {
	NewVaultKeyType       string `json:"new_vault_key_type"`
	NewSalt               string `json:"new_salt"`
	NewAuthKeyHash        string `json:"new_auth_key_hash"`
	NewWrappedDEK         string `json:"new_wrapped_dek"`
	NewWrappedPrivateKey  string `json:"new_wrapped_private_key"`
	NewRecoveryWrappedDEK string `json:"new_recovery_wrapped_dek"`
}

// @Summary		Complete trusted contact recovery
// @Tags			recovery
// @Accept			json
// @Produce		json
// @Param			id		path	string					true	"Recovery request ID"
// @Param			body	body	CompleteRecoveryRequest	true	"New crypto material"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/recovery/request/{id}/complete [post]
func (h *RecoveryHandler) CompleteRecovery(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

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

	var rr model.RecoveryRequests
	if err := SELECT(table.RecoveryRequests.UserID, table.RecoveryRequests.Status).
		FROM(table.RecoveryRequests).WHERE(
		table.RecoveryRequests.ID.EQ(UUID(requestID)).
			AND(table.RecoveryRequests.UserID.EQ(UUID(userID))),
	).Query(h.db, &rr); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "recovery request not found"})
		return
	}
	if rr.Status != "approved" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "request is not approved"})
		return
	}

	newSalt, _ := base64.StdEncoding.DecodeString(req.NewSalt)
	newAuthKeyHash, _ := base64.StdEncoding.DecodeString(req.NewAuthKeyHash)
	newWrappedDEK, _ := base64.StdEncoding.DecodeString(req.NewWrappedDEK)
	newWrappedPrivateKey, _ := base64.StdEncoding.DecodeString(req.NewWrappedPrivateKey)
	var newRecoveryWrappedDEK []byte
	if req.NewRecoveryWrappedDEK != "" {
		newRecoveryWrappedDEK, _ = base64.StdEncoding.DecodeString(req.NewRecoveryWrappedDEK)
	}

	now := time.Now().UTC()

	// identity_id = users.id, so filter by rr.UserID
	if _, err := table.Identities.UPDATE(
		table.Identities.VaultKeyType,
		table.Identities.Salt,
		table.Identities.AuthKeyHash,
		table.Identities.WrappedDek,
		table.Identities.WrappedPrivateKey,
		table.Identities.RecoveryWrappedDek,
		table.Identities.UpdatedAt,
	).SET(
		req.NewVaultKeyType,
		newSalt,
		newAuthKeyHash,
		newWrappedDEK,
		newWrappedPrivateKey,
		newRecoveryWrappedDEK,
		now,
	).WHERE(table.Identities.IdentityID.EQ(UUID(rr.UserID))).Exec(h.db); err != nil {
		slog.Error("complete-recovery: update identity", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update vault"})
		return
	}

	if _, err := table.RecoveryRequests.UPDATE(
		table.RecoveryRequests.Status,
		table.RecoveryRequests.CompletedAt,
	).SET("completed", now).
		WHERE(table.RecoveryRequests.ID.EQ(UUID(requestID))).
		Exec(h.db); err != nil {
		slog.Error("complete-recovery: update request", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to complete request"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "vault recovered"})
}

// --- Public Key Lookup ---

type PublicKeyResponse struct {
	Email     string `json:"email"`
	PublicKey string `json:"public_key"`
}

// @Summary		Lookup user public key
// @Tags			recovery
// @Produce		json
// @Param			email	query	string	true	"User email"
// @Success		200		{object}	PublicKeyResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/users/public-key [get]
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

	aid, err := user_lookup.IDByEmail(r.Context(), h.db, email)
	if err != nil {
		if errors.Is(err, user_lookup.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
		slog.Error("public-key: resolve email", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to resolve user"})
		return
	}

	var identity model.Identities
	if err := SELECT(table.Identities.PublicKey).FROM(table.Identities).
		WHERE(table.Identities.IdentityID.EQ(UUID(aid))).
		Query(h.db, &identity); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, PublicKeyResponse{
		Email:     email,
		PublicKey: base64.StdEncoding.EncodeToString(identity.PublicKey),
	})
}
