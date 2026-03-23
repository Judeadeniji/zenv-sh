package handler

import (
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// AuthHandler handles vault setup, unlock, and user state endpoints.
// Identity (signup, login, logout) is handled by the standalone auth server.
type AuthHandler struct {
	db       *sql.DB
	identity *middleware.IdentitySession
}

func NewAuthHandler(db *sql.DB, identity *middleware.IdentitySession) *AuthHandler {
	return &AuthHandler{db: db, identity: identity}
}

// --- Vault Unlock ---

type UnlockRequest struct {
	AuthKeyHash string `json:"auth_key_hash"` // base64
}

type UnlockResponse struct {
	WrappedDEK        string `json:"wrapped_dek"`         // base64
	WrappedPrivateKey string `json:"wrapped_private_key"` // base64
	PublicKey         string `json:"public_key"`          // base64
}

// Unlock verifies the Vault Key (via Auth Key hash) and returns the Wrapped DEK.
// Requires an active identity session.
//
//	@Summary		Unlock vault
//	@Description	Verify Auth Key hash (Vault Key proof). Returns wrapped DEK + keypair on success.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		UnlockRequest	true	"Auth Key hash (base64)"
//	@Success		200		{object}	UnlockResponse
//	@Failure		403		{object}	ErrorResponse	"Wrong Vault Key"
//	@Security		SessionAuth
//	@Router			/auth/unlock [post]
func (h *AuthHandler) Unlock(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	var req UnlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	submittedHash, err := base64.StdEncoding.DecodeString(req.AuthKeyHash)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in auth_key_hash"})
		return
	}

	// Fetch user by identity ID.
	var user model.Users
	stmt := SELECT(
		table.Users.AuthKeyHash,
		table.Users.WrappedDek,
		table.Users.WrappedPrivateKey,
		table.Users.PublicKey,
	).FROM(table.Users).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))

	err = stmt.Query(h.db, &user)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		slog.Error("unlock: fetch user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch user"})
		return
	}

	// Constant-time comparison to prevent timing attacks.
	// The client already ran HashAuthKey(authKey) — we compare directly.
	if subtle.ConstantTimeCompare(submittedHash, user.AuthKeyHash) != 1 {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "wrong Vault Key"})
		return
	}

	// Auth Key verified — mark vault as unlocked in Redis.
	// Use the token part of the cookie (before ".") to match how RequireSession looks it up.
	cookie, _ := r.Cookie(middleware.IdentitySessionCookie)
	if cookie != nil {
		sessionToken := strings.Split(cookie.Value, ".")[0]
		if err := h.identity.SetVaultUnlocked(r.Context(), sessionToken, time.Now().Add(24*time.Hour)); err != nil {
			slog.Error("unlock: set vault state", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, UnlockResponse{
		WrappedDEK:        base64.StdEncoding.EncodeToString(user.WrappedDek),
		WrappedPrivateKey: base64.StdEncoding.EncodeToString(user.WrappedPrivateKey),
		PublicKey:         base64.StdEncoding.EncodeToString(user.PublicKey),
	})
}

// --- Vault Setup ---

type SetupVaultRequest struct {
	VaultKeyType       string `json:"vault_key_type"`       // "pin" or "passphrase"
	Salt               string `json:"salt"`                 // base64
	AuthKeyHash        string `json:"auth_key_hash"`        // base64
	WrappedDEK         string `json:"wrapped_dek"`          // base64
	PublicKey          string `json:"public_key"`           // base64
	WrappedPrivateKey  string `json:"wrapped_private_key"`  // base64
	RecoveryWrappedDEK string `json:"recovery_wrapped_dek"` // base64, optional — DEK wrapped with recovery key
	RecoveryDisabled   bool   `json:"recovery_disabled"`    // enterprise opt-in to disable recovery
}

type SetupVaultResponse struct {
	UserID             string `json:"user_id"`
	VaultSetupComplete bool   `json:"vault_setup_complete"`
}

// SetupVault creates a zEnv user row linked to the authenticated identity.
// Called after the user signs in and chooses a Vault Key.
//
// @Summary		Setup vault
// @Description	Store client-generated crypto material and link to authenticated identity.
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			body	body		SetupVaultRequest	true	"Crypto material from client"
// @Success		201		{object}	SetupVaultResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Security		SessionAuth
// @Router			/auth/setup-vault [post]
func (h *AuthHandler) SetupVault(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.IdentityID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var req SetupVaultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Salt == "" || req.AuthKeyHash == "" || req.WrappedDEK == "" || req.PublicKey == "" || req.WrappedPrivateKey == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "all fields are required"})
		return
	}
	if req.VaultKeyType != "pin" && req.VaultKeyType != "passphrase" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "vault_key_type must be 'pin' or 'passphrase'"})
		return
	}

	// Check if vault already set up for this user.
	var existingID string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id FROM users WHERE identity_id = $1`, sess.IdentityID,
	).Scan(&existingID)
	if err == nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "vault already set up for this account"})
		return
	}

	// Decode base64 fields.
	salt, err := base64.StdEncoding.DecodeString(req.Salt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in salt"})
		return
	}
	authKeyHash, err := base64.StdEncoding.DecodeString(req.AuthKeyHash)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in auth_key_hash"})
		return
	}
	wrappedDEK, err := base64.StdEncoding.DecodeString(req.WrappedDEK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in wrapped_dek"})
		return
	}
	publicKey, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in public_key"})
		return
	}
	wrappedPrivateKey, err := base64.StdEncoding.DecodeString(req.WrappedPrivateKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in wrapped_private_key"})
		return
	}

	// Decode optional recovery material.
	var recoveryWrappedDEK []byte
	if req.RecoveryWrappedDEK != "" {
		recoveryWrappedDEK, err = base64.StdEncoding.DecodeString(req.RecoveryWrappedDEK)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in recovery_wrapped_dek"})
			return
		}
	}

	userID := uuid.New()
	now := time.Now().UTC()

	insertStmt := table.Users.INSERT(
		table.Users.ID,
		table.Users.Email,
		table.Users.AuthKeyHash,
		table.Users.VaultKeyType,
		table.Users.Salt,
		table.Users.WrappedDek,
		table.Users.PublicKey,
		table.Users.WrappedPrivateKey,
		table.Users.IdentityID,
		table.Users.RecoveryWrappedDek,
		table.Users.RecoveryDisabled,
		table.Users.CreatedAt,
		table.Users.UpdatedAt,
	).VALUES(
		userID,
		sess.Email,
		authKeyHash,
		req.VaultKeyType,
		salt,
		wrappedDEK,
		publicKey,
		wrappedPrivateKey,
		sess.IdentityID,
		recoveryWrappedDEK,
		req.RecoveryDisabled,
		now,
		now,
	)

	if _, err := insertStmt.Exec(h.db); err != nil {
		slog.Error("setup-vault: insert user", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to set up vault"})
		return
	}

	// Mark vault as unlocked — the user just derived keys during setup.
	cookie, _ := r.Cookie(middleware.IdentitySessionCookie)
	if cookie != nil {
		sessionToken := strings.Split(cookie.Value, ".")[0]
		if err := h.identity.SetVaultUnlocked(r.Context(), sessionToken, time.Now().Add(24*time.Hour)); err != nil {
			slog.Error("setup-vault: set vault unlocked", "error", err)
		}
	}

	writeJSON(w, http.StatusCreated, SetupVaultResponse{
		UserID:             userID.String(),
		VaultSetupComplete: true,
	})
}

// --- Me (user state) ---

type MeResponse struct {
	Email              string `json:"email"`
	Name               string `json:"name,omitempty"`
	VaultSetupComplete bool   `json:"vault_setup_complete"`
	VaultKeyType       string `json:"vault_key_type,omitempty"`
	Salt               string `json:"salt,omitempty"` // base64
	VaultUnlocked      bool   `json:"vault_unlocked"`
}

// Me returns the current user's auth state.
//
// @Summary		Get auth state
// @Description	Returns identity, vault setup status, and vault lock state.
// @Tags			auth
// @Produce		json
// @Success		200	{object}	MeResponse
// @Security		SessionAuth
// @Router			/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.IdentityID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	resp := MeResponse{
		Email:         sess.Email,
		VaultUnlocked: sess.IsVaultUnlocked(),
	}

	// Fetch display name from identity provider's user table.
	var name string
	_ = h.db.QueryRowContext(r.Context(),
		`SELECT name FROM "user" WHERE id = $1`, sess.IdentityID,
	).Scan(&name)
	resp.Name = name

	// Check if zEnv user exists (vault set up).
	var user model.Users
	stmt := SELECT(
		table.Users.VaultKeyType,
		table.Users.Salt,
	).FROM(table.Users).WHERE(
		table.Users.IdentityID.EQ(String(sess.IdentityID)),
	)

	if err := stmt.Query(h.db, &user); err == nil {
		resp.VaultSetupComplete = true
		resp.VaultKeyType = user.VaultKeyType
		resp.Salt = base64.StdEncoding.EncodeToString(user.Salt)
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Vault Key Change ---

type ChangeVaultKeyRequest struct {
	CurrentAuthKeyHash   string `json:"current_auth_key_hash"`   // base64
	NewVaultKeyType      string `json:"new_vault_key_type"`      // "pin" or "passphrase"
	NewSalt              string `json:"new_salt"`                // base64
	NewAuthKeyHash       string `json:"new_auth_key_hash"`       // base64
	NewWrappedDEK        string `json:"new_wrapped_dek"`         // base64
	NewWrappedPrivateKey string `json:"new_wrapped_private_key"` // base64
}

// ChangeVaultKey rotates the Vault Key without touching any item rows.
// The client derives the old KEK, unwraps the DEK, derives a new KEK from
// the new Vault Key, re-wraps the same DEK, and sends the new crypto material.
// This is an O(1) operation — zero item rows are modified.
//
//	@Summary		Change vault key
//	@Description	Rotate vault key: verify current auth key, store new crypto material. O(1) — no item rows touched.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ChangeVaultKeyRequest	true	"Current auth proof + new crypto material"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse	"Wrong current Vault Key"
//	@Security		SessionAuth
//	@Router			/auth/change-vault-key [put]
func (h *AuthHandler) ChangeVaultKey(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}

	var req ChangeVaultKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	// Validate required fields.
	if req.CurrentAuthKeyHash == "" || req.NewSalt == "" || req.NewAuthKeyHash == "" ||
		req.NewWrappedDEK == "" || req.NewWrappedPrivateKey == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "all fields are required"})
		return
	}
	if req.NewVaultKeyType != "pin" && req.NewVaultKeyType != "passphrase" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "new_vault_key_type must be 'pin' or 'passphrase'"})
		return
	}

	// Decode current auth key hash.
	currentHash, err := base64.StdEncoding.DecodeString(req.CurrentAuthKeyHash)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in current_auth_key_hash"})
		return
	}

	// Fetch stored auth key hash for verification.
	var user model.Users
	stmt := SELECT(
		table.Users.ID,
		table.Users.AuthKeyHash,
	).FROM(table.Users).WHERE(table.Users.IdentityID.EQ(String(sess.IdentityID)))

	if err := stmt.Query(h.db, &user); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "vault not set up"})
			return
		}
		slog.Error("change-vault-key: fetch user", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch user"})
		return
	}

	// Verify current Vault Key via Auth Key hash (same as unlock flow).
	rehashedSubmitted := amnesia.HashAuthKey(currentHash)
	if subtle.ConstantTimeCompare(rehashedSubmitted, user.AuthKeyHash) != 1 {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "wrong current Vault Key"})
		return
	}

	// Decode new crypto material.
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

	// Update user row with new crypto material. Same DEK, just re-wrapped.
	now := time.Now().UTC()
	updateStmt := table.Users.UPDATE(
		table.Users.VaultKeyType,
		table.Users.Salt,
		table.Users.AuthKeyHash,
		table.Users.WrappedDek,
		table.Users.WrappedPrivateKey,
		table.Users.UpdatedAt,
	).SET(
		req.NewVaultKeyType,
		newSalt,
		newAuthKeyHash,
		newWrappedDEK,
		newWrappedPrivateKey,
		now,
	).WHERE(table.Users.ID.EQ(UUID(user.ID)))

	if _, err := updateStmt.Exec(h.db); err != nil {
		slog.Error("change-vault-key: update user", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update vault key"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "vault key changed"})
}

// ErrorResponse is returned on all error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
