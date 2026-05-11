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

// ErrorResponse is returned on all error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Vault Lock ---

// Lock clears the vault unlock flag for the current session in Redis.
//
//	@Summary        Lock vault
//	@Description    Clears the vault unlock record in Redis, requiring the user to re-enter their Vault Key on next access. Safe to call even if already locked.
//	@Tags           auth
//	@Produce        json
//	@Success        204 "Vault locked successfully"
//	@Failure        401 {object}    ErrorResponse   "No active session"
//	@Security       SessionAuth
//	@Router         /auth/lock [post]
func (h *AuthHandler) Lock(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(middleware.IdentitySessionCookie)
	if err != nil || cookie.Value == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	sessionToken := strings.Split(cookie.Value, ".")[0]
	if err := h.identity.ClearVaultUnlocked(r.Context(), sessionToken); err != nil {
		slog.Error("lock: clear vault state", "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Vault Unlock ---

// UnlockRequest is the request body for POST /auth/unlock.
type UnlockRequest struct {
	// AuthKeyHash is the result of HashAuthKey(vaultKey), base64-encoded.
	// Never send the raw Vault Key — only the derived hash.
	AuthKeyHash string `json:"auth_key_hash" example:"base64encodedstring=="`
}

// UnlockResponse contains the encrypted key material returned after a successful unlock.
type UnlockResponse struct {
	// WrappedDEK is the Data Encryption Key wrapped with the Key Encryption Key, base64-encoded.
	WrappedDEK string `json:"wrapped_dek" example:"base64encodedstring=="`
	// WrappedPrivateKey is the user's Ed25519 private key wrapped with the DEK, base64-encoded.
	WrappedPrivateKey string `json:"wrapped_private_key" example:"base64encodedstring=="`
	// PublicKey is the user's raw Ed25519 public key, base64-encoded.
	PublicKey string `json:"public_key" example:"base64encodedstring=="`
}

// Unlock verifies the Vault Key (via Auth Key hash) and returns the Wrapped DEK.
//
//	@Summary        Unlock vault
//	@Description    Verify the Auth Key hash (proof of Vault Key knowledge). On success, marks the session as vault-unlocked in Redis and returns the wrapped DEK and keypair so the client can derive the KEK and unwrap locally. The raw Vault Key never leaves the client.
//	@Tags           auth
//	@Accept         json
//	@Produce        json
//	@Param          body    body        UnlockRequest   true    "Auth Key hash derived from the user's Vault Key"
//	@Success        200     {object}    UnlockResponse  "Vault unlocked — key material returned"
//	@Failure        400     {object}    ErrorResponse   "Invalid request body or malformed base64"
//	@Failure        401     {object}    ErrorResponse   "No active session"
//	@Failure        403     {object}    ErrorResponse   "Wrong Vault Key — auth key hash mismatch"
//	@Failure        404     {object}    ErrorResponse   "Vault not set up for this account"
//	@Failure        500     {object}    ErrorResponse   "Internal server error"
//	@Security       SessionAuth
//	@Router         /auth/unlock [post]
func (h *AuthHandler) Unlock(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var req UnlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	submittedHash, err := decodeBase64Field(req.AuthKeyHash, "auth_key_hash")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in auth_key_hash"})
		return
	}

	var identity model.Identities
	if err := SELECT(
		table.Identities.AuthKeyHash,
		table.Identities.WrappedDek,
		table.Identities.WrappedPrivateKey,
		table.Identities.PublicKey,
	).FROM(table.Identities).WHERE(table.Identities.IdentityID.EQ(UUID(userID))).
		Query(h.db, &identity); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "vault not set up"})
			return
		}
		slog.Error("unlock: fetch identity", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch user"})
		return
	}

	// Constant-time comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare(submittedHash, identity.AuthKeyHash) != 1 {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "wrong Vault Key"})
		return
	}

	cookie, _ := r.Cookie(middleware.IdentitySessionCookie)
	if cookie != nil {
		sessionToken := strings.Split(cookie.Value, ".")[0]
		if err := h.identity.SetVaultUnlocked(r.Context(), sessionToken, time.Now().Add(24*time.Hour)); err != nil {
			slog.Error("unlock: set vault state", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, UnlockResponse{
		WrappedDEK:        base64.StdEncoding.EncodeToString(identity.WrappedDek),
		WrappedPrivateKey: base64.StdEncoding.EncodeToString(identity.WrappedPrivateKey),
		PublicKey:         base64.StdEncoding.EncodeToString(identity.PublicKey),
	})
}

// --- Vault Setup ---

// SetupVaultRequest is the request body for POST /auth/setup-vault.
type SetupVaultRequest struct {
	// VaultKeyType is the type of Vault Key the user chose. Must be "pin" or "passphrase".
	VaultKeyType string `json:"vault_key_type" enums:"pin,passphrase" example:"passphrase"`
	// Salt is the random salt used to derive the Key Encryption Key from the Vault Key, base64-encoded.
	Salt string `json:"salt" example:"base64encodedstring=="`
	// AuthKeyHash is HashAuthKey(vaultKey), used to verify the Vault Key on unlock, base64-encoded.
	AuthKeyHash string `json:"auth_key_hash" example:"base64encodedstring=="`
	// WrappedDEK is the Data Encryption Key wrapped with the Key Encryption Key, base64-encoded.
	WrappedDEK string `json:"wrapped_dek" example:"base64encodedstring=="`
	// PublicKey is the user's raw Ed25519 public key, base64-encoded.
	PublicKey string `json:"public_key" example:"base64encodedstring=="`
	// WrappedPrivateKey is the user's Ed25519 private key wrapped with the DEK, base64-encoded.
	WrappedPrivateKey string `json:"wrapped_private_key" example:"base64encodedstring=="`
	// RecoveryWrappedDEK is the DEK wrapped with the recovery key, base64-encoded. Optional.
	RecoveryWrappedDEK string `json:"recovery_wrapped_dek,omitempty" example:"base64encodedstring=="`
	// RecoveryDisabled disables all recovery methods for this account. Enterprise opt-in.
	RecoveryDisabled bool `json:"recovery_disabled" example:"false"`
}

// SetupVaultResponse is returned after a successful vault setup.
type SetupVaultResponse struct {
	// UserID is the newly created vault identity UUID.
	UserID string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// VaultSetupComplete is always true on success.
	VaultSetupComplete bool `json:"vault_setup_complete" example:"true"`
}

// SetupVault creates a vault identity linked to the authenticated user.
//
//	@Summary        Setup vault
//	@Description    Store client-generated cryptographic material and link to the authenticated identity. Must be called once after signup before any vault operations. All crypto material is generated client-side — the server stores ciphertext only. On success, the session is immediately marked as vault-unlocked.
//	@Tags           auth
//	@Accept         json
//	@Produce        json
//	@Param          body    body        SetupVaultRequest   true    "Client-generated crypto material"
//	@Success        201     {object}    SetupVaultResponse  "Vault created and session unlocked"
//	@Failure        400     {object}    ErrorResponse       "Missing required fields, invalid vault_key_type, or malformed base64"
//	@Failure        401     {object}    ErrorResponse       "No active session"
//	@Failure        409     {object}    ErrorResponse       "Vault already set up for this account"
//	@Failure        500     {object}    ErrorResponse       "Internal server error"
//	@Security       SessionAuth
//	@Router         /auth/setup-vault [post]
func (h *AuthHandler) SetupVault(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var req SetupVaultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	if req.Salt == "" || req.AuthKeyHash == "" || req.WrappedDEK == "" || req.PublicKey == "" || req.WrappedPrivateKey == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "salt, auth_key_hash, wrapped_dek, public_key, and wrapped_private_key are required"})
		return
	}
	if req.VaultKeyType != "pin" && req.VaultKeyType != "passphrase" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "vault_key_type must be 'pin' or 'passphrase'"})
		return
	}

	// Use HasIdentity from session (set by LEFT JOIN in RequireSession) to avoid a round trip.
	if sess.HasIdentity {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "vault already set up for this account"})
		return
	}

	salt, err := decodeBase64Field(req.Salt, "salt")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in salt"})
		return
	}
	authKeyHash, err := decodeBase64Field(req.AuthKeyHash, "auth_key_hash")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in auth_key_hash"})
		return
	}
	wrappedDEK, err := decodeBase64Field(req.WrappedDEK, "wrapped_dek")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in wrapped_dek"})
		return
	}
	publicKey, err := decodeBase64Field(req.PublicKey, "public_key")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in public_key"})
		return
	}
	wrappedPrivateKey, err := decodeBase64Field(req.WrappedPrivateKey, "wrapped_private_key")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in wrapped_private_key"})
		return
	}
	var recoveryWrappedDEK []byte
	if req.RecoveryWrappedDEK != "" {
		recoveryWrappedDEK, err = decodeBase64Field(req.RecoveryWrappedDEK, "recovery_wrapped_dek")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in recovery_wrapped_dek"})
			return
		}
	}

	identityID := uuid.New()
	now := time.Now().UTC()

	if _, err := table.Identities.INSERT(
		table.Identities.ID,
		table.Identities.AuthKeyHash,
		table.Identities.VaultKeyType,
		table.Identities.Salt,
		table.Identities.WrappedDek,
		table.Identities.PublicKey,
		table.Identities.WrappedPrivateKey,
		table.Identities.IdentityID,
		table.Identities.RecoveryWrappedDek,
		table.Identities.RecoveryDisabled,
		table.Identities.CreatedAt,
		table.Identities.UpdatedAt,
	).VALUES(
		identityID,
		authKeyHash,
		req.VaultKeyType,
		salt,
		wrappedDEK,
		publicKey,
		wrappedPrivateKey,
		userID,
		recoveryWrappedDEK,
		req.RecoveryDisabled,
		now,
		now,
	).Exec(h.db); err != nil {
		slog.Error("setup-vault: insert identity", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to set up vault"})
		return
	}

	cookie, _ := r.Cookie(middleware.IdentitySessionCookie)
	if cookie != nil {
		sessionToken := strings.Split(cookie.Value, ".")[0]
		if err := h.identity.SetVaultUnlocked(r.Context(), sessionToken, time.Now().Add(24*time.Hour)); err != nil {
			slog.Error("setup-vault: set vault unlocked", "error", err)
		}
	}

	writeJSON(w, http.StatusCreated, SetupVaultResponse{
		UserID:             identityID.String(),
		VaultSetupComplete: true,
	})
}

// --- Me ---

// MeResponse is the response body for GET /auth/me.
type MeResponse struct {
	// Email is the user's email address from their identity provider.
	Email string `json:"email" example:"user@example.com"`
	// Name is the user's display name from their identity provider.
	Name string `json:"name,omitempty" example:"Jane Doe"`
	// VaultSetupComplete is true if the user has completed vault setup.
	VaultSetupComplete bool `json:"vault_setup_complete" example:"true"`
	// VaultKeyType is the type of Vault Key the user chose ("pin" or "passphrase"). Only present if vault is set up.
	VaultKeyType model.VaultKeyType `json:"vault_key_type,omitempty" enums:"pin,passphrase" example:"passphrase"`
	// Salt is the KDF salt for the user's Key Encryption Key, base64-encoded. Only present if vault is set up.
	Salt string `json:"salt,omitempty" example:"base64encodedstring=="`
	// VaultUnlocked is true if the user has completed both auth layers in this session.
	VaultUnlocked bool `json:"vault_unlocked" example:"false"`
}

// Me returns the current user's auth and vault state.
//
//	@Summary        Get auth state
//	@Description    Returns the authenticated user's identity info, vault setup status, and vault lock state. Use this to determine whether to show the vault setup flow, the unlock prompt, or the main UI. The salt is returned so the client can derive the KEK locally without an extra round trip.
//	@Tags           auth
//	@Produce        json
//	@Success        200 {object}    MeResponse      "Current user state"
//	@Failure        401 {object}    ErrorResponse   "No active session"
//	@Security       SessionAuth
//	@Router         /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	// Fetch name from unified users table.
	var user model.Users
	_ = SELECT(table.Users.Name).FROM(table.Users).WHERE(table.Users.ID.EQ(UUID(userID))).
		Query(h.db, &user)

	resp := MeResponse{
		Email:         sess.Email,
		Name:          user.Name,
		VaultUnlocked: sess.IsVaultUnlocked(),
	}

	// Fetch vault material if identity exists.
	var identity model.Identities
	if err := SELECT(
		table.Identities.VaultKeyType,
		table.Identities.Salt,
	).FROM(table.Identities).WHERE(table.Identities.IdentityID.EQ(UUID(userID))).
		Query(h.db, &identity); err == nil {
		resp.VaultSetupComplete = true
		resp.VaultKeyType = identity.VaultKeyType
		resp.Salt = base64.StdEncoding.EncodeToString(identity.Salt)
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Vault Key Change ---

// ChangeVaultKeyRequest is the request body for PUT /auth/change-vault-key.
type ChangeVaultKeyRequest struct {
	// CurrentAuthKeyHash is HashAuthKey(currentVaultKey), used to verify the current Vault Key before rotation, base64-encoded.
	CurrentAuthKeyHash string `json:"current_auth_key_hash" example:"base64encodedstring=="`
	// NewVaultKeyType is the type of the new Vault Key. Must be "pin" or "passphrase".
	NewVaultKeyType string `json:"new_vault_key_type" enums:"pin,passphrase" example:"passphrase"`
	// NewSalt is a freshly generated KDF salt for the new Key Encryption Key, base64-encoded.
	NewSalt string `json:"new_salt" example:"base64encodedstring=="`
	// NewAuthKeyHash is HashAuthKey(newVaultKey), base64-encoded.
	NewAuthKeyHash string `json:"new_auth_key_hash" example:"base64encodedstring=="`
	// NewWrappedDEK is the same DEK re-wrapped with the new KEK, base64-encoded.
	NewWrappedDEK string `json:"new_wrapped_dek" example:"base64encodedstring=="`
	// NewWrappedPrivateKey is the private key re-wrapped with the new DEK, base64-encoded.
	NewWrappedPrivateKey string `json:"new_wrapped_private_key" example:"base64encodedstring=="`
}

// ChangeVaultKey rotates the Vault Key without touching any vault item rows.
//
//	@Summary        Change vault key
//	@Description    Rotate the Vault Key. The client derives the old KEK, unwraps the DEK, derives a new KEK from the new Vault Key, re-wraps the same DEK, and submits the new crypto material. This is an O(1) operation — zero vault item rows are touched. Requires the current Vault Key to be verified before rotation is applied.
//	@Tags           auth
//	@Accept         json
//	@Produce        json
//	@Param          body    body        ChangeVaultKeyRequest   true    "Current auth proof and new crypto material"
//	@Success        200     {object}    map[string]string       "Vault key rotated successfully"
//	@Failure        400     {object}    ErrorResponse           "Missing required fields, invalid vault_key_type, or malformed base64"
//	@Failure        401     {object}    ErrorResponse           "No active session"
//	@Failure        403     {object}    ErrorResponse           "Wrong current Vault Key"
//	@Failure        404     {object}    ErrorResponse           "Vault not set up"
//	@Failure        500     {object}    ErrorResponse           "Internal server error"
//	@Security       SessionAuth
//	@Router         /auth/change-vault-key [put]
func (h *AuthHandler) ChangeVaultKey(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var req ChangeVaultKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	if req.CurrentAuthKeyHash == "" || req.NewSalt == "" || req.NewAuthKeyHash == "" ||
		req.NewWrappedDEK == "" || req.NewWrappedPrivateKey == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "all fields are required"})
		return
	}
	if req.NewVaultKeyType != "pin" && req.NewVaultKeyType != "passphrase" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "new_vault_key_type must be 'pin' or 'passphrase'"})
		return
	}

	currentHash, err := decodeBase64Field(req.CurrentAuthKeyHash, "current_auth_key_hash")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in current_auth_key_hash"})
		return
	}

	var identity model.Identities
	if err := SELECT(table.Identities.ID, table.Identities.AuthKeyHash).
		FROM(table.Identities).WHERE(table.Identities.IdentityID.EQ(UUID(userID))).
		Query(h.db, &identity); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "vault not set up"})
			return
		}
		slog.Error("change-vault-key: fetch identity", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch user"})
		return
	}

	rehashedSubmitted := amnesia.HashAuthKey(currentHash)
	if subtle.ConstantTimeCompare(rehashedSubmitted, identity.AuthKeyHash) != 1 {
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "wrong current Vault Key"})
		return
	}

	newSalt, err := decodeBase64Field(req.NewSalt, "new_salt")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_salt"})
		return
	}
	newAuthKeyHash, err := decodeBase64Field(req.NewAuthKeyHash, "new_auth_key_hash")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_auth_key_hash"})
		return
	}
	newWrappedDEK, err := decodeBase64Field(req.NewWrappedDEK, "new_wrapped_dek")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_wrapped_dek"})
		return
	}
	newWrappedPrivateKey, err := decodeBase64Field(req.NewWrappedPrivateKey, "new_wrapped_private_key")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64 in new_wrapped_private_key"})
		return
	}

	if _, err := table.Identities.UPDATE(
		table.Identities.VaultKeyType,
		table.Identities.Salt,
		table.Identities.AuthKeyHash,
		table.Identities.WrappedDek,
		table.Identities.WrappedPrivateKey,
		table.Identities.UpdatedAt,
	).SET(
		req.NewVaultKeyType,
		newSalt,
		newAuthKeyHash,
		newWrappedDEK,
		newWrappedPrivateKey,
		time.Now().UTC(),
	).WHERE(table.Identities.ID.EQ(UUID(identity.ID))).Exec(h.db); err != nil {
		slog.Error("change-vault-key: update identity", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update vault key"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "vault key changed"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func decodeBase64Field(s, field string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New("invalid base64 in " + field)
	}
	return b, nil
}
