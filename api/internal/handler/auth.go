package handler

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// AuthHandler handles identity + encryption layer auth endpoints.
type AuthHandler struct {
	db      *sql.DB
	session *middleware.SessionManager
	ba      *middleware.BetterAuthSession
}

func NewAuthHandler(db *sql.DB, session *middleware.SessionManager, ba *middleware.BetterAuthSession) *AuthHandler {
	return &AuthHandler{db: db, session: session, ba: ba}
}

// --- Signup ---

type SignupRequest struct {
	Email             string `json:"email"`
	VaultKeyType      string `json:"vault_key_type"`      // "pin" or "passphrase"
	Salt              string `json:"salt"`                // base64
	AuthKeyHash       string `json:"auth_key_hash"`       // base64
	WrappedDEK        string `json:"wrapped_dek"`         // base64
	PublicKey         string `json:"public_key"`          // base64
	WrappedPrivateKey string `json:"wrapped_private_key"` // base64
}

type SignupResponse struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

// Signup creates a new account with all crypto material from the client.
// The server stores only ciphertext and hashes — it cannot derive KEK or unwrap the DEK.
//
//	@Summary		Create account
//	@Description	Register with client-generated crypto material. Server stores ciphertext only.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		SignupRequest	true	"Crypto material from client"
//	@Success		201		{object}	SignupResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Router			/auth/signup [post]
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Salt == "" || req.AuthKeyHash == "" || req.WrappedDEK == "" || req.PublicKey == "" || req.WrappedPrivateKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "all fields are required"})
		return
	}

	if req.VaultKeyType != "pin" && req.VaultKeyType != "passphrase" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "vault_key_type must be 'pin' or 'passphrase'"})
		return
	}

	// Decode base64 fields.
	salt, err := base64.StdEncoding.DecodeString(req.Salt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in salt"})
		return
	}
	authKeyHash, err := base64.StdEncoding.DecodeString(req.AuthKeyHash)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in auth_key_hash"})
		return
	}
	wrappedDEK, err := base64.StdEncoding.DecodeString(req.WrappedDEK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in wrapped_dek"})
		return
	}
	publicKey, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in public_key"})
		return
	}
	wrappedPrivateKey, err := base64.StdEncoding.DecodeString(req.WrappedPrivateKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 in wrapped_private_key"})
		return
	}

	// Check if email already exists.
	var existing model.Users
	existsStmt := SELECT(table.Users.ID).FROM(table.Users).WHERE(table.Users.Email.EQ(String(req.Email)))
	err = existsStmt.Query(h.db, &existing)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "account already exists for this email"})
		return
	}

	// Insert user.
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
		table.Users.CreatedAt,
		table.Users.UpdatedAt,
	).VALUES(
		userID,
		req.Email,
		authKeyHash,
		req.VaultKeyType,
		salt,
		wrappedDEK,
		publicKey,
		wrappedPrivateKey,
		now,
		now,
	)

	_, err = insertStmt.Exec(h.db)
	if err != nil {
		slog.Error("signup: insert user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create account"})
		return
	}

	// Create session (identity layer complete).
	sess, err := h.session.Create(r.Context(), userID.String(), req.Email)
	if err != nil {
		slog.Error("signup: create session", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	// Mark vault as unlocked (user just set their Vault Key — they have the KEK).
	unlockTime := time.Now().UTC().Format(time.RFC3339)
	sess.VaultUnlockedAt = &unlockTime
	h.session.Update(r.Context(), sess)

	middleware.SetSessionCookie(w, sess.ID)
	writeJSON(w, http.StatusCreated, SignupResponse{
		UserID:    userID.String(),
		SessionID: sess.ID,
	})
}

// --- Dev Login (temporary — replaced by OAuth in production) ---

type DevLoginRequest struct {
	Email string `json:"email"`
}

type DevLoginResponse struct {
	UserID       string `json:"user_id"`
	SessionID    string `json:"session_id"`
	VaultKeyType string `json:"vault_key_type"`
	Salt         string `json:"salt"` // base64 — client needs this to re-derive KEK
}

// DevLogin creates a session for an existing user by email.
// Development only — in production this is handled by OAuth callback.
//
//	@Summary		Dev login (temporary)
//	@Description	Create session by email. Development only — replaced by OAuth in production.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		DevLoginRequest	true	"Email to login"
//	@Success		200		{object}	DevLoginResponse
//	@Failure		404		{object}	ErrorResponse
//	@Router			/auth/login [post]
func (h *AuthHandler) DevLogin(w http.ResponseWriter, r *http.Request) {
	var req DevLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	var user model.Users
	stmt := SELECT(
		table.Users.ID,
		table.Users.Email,
		table.Users.VaultKeyType,
		table.Users.Salt,
	).FROM(table.Users).WHERE(table.Users.Email.EQ(String(req.Email)))

	err := stmt.Query(h.db, &user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no account found for this email"})
			return
		}
		slog.Error("dev-login: fetch user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch user"})
		return
	}

	sess, err := h.session.Create(r.Context(), user.ID.String(), user.Email)
	if err != nil {
		slog.Error("dev-login: create session", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	middleware.SetSessionCookie(w, sess.ID)
	writeJSON(w, http.StatusOK, DevLoginResponse{
		UserID:       user.ID.String(),
		SessionID:    sess.ID,
		VaultKeyType: user.VaultKeyType,
		Salt:         base64.StdEncoding.EncodeToString(user.Salt),
	})
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
// Requires an active session (identity layer must pass first).
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

	// Fetch user — by zEnv user ID or BA user ID depending on session type.
	var user model.Users
	var stmt SelectStatement

	if sess.UserID != "" {
		uid, parseErr := uuid.Parse(sess.UserID)
		if parseErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid session"})
			return
		}
		stmt = SELECT(
			table.Users.AuthKeyHash,
			table.Users.WrappedDek,
			table.Users.WrappedPrivateKey,
			table.Users.PublicKey,
		).FROM(table.Users).WHERE(table.Users.ID.EQ(UUID(uid)))
	} else if sess.BAUserID != "" {
		stmt = SELECT(
			table.Users.AuthKeyHash,
			table.Users.WrappedDek,
			table.Users.WrappedPrivateKey,
			table.Users.PublicKey,
		).FROM(table.Users).WHERE(table.Users.BetterAuthUserID.EQ(String(sess.BAUserID)))
	} else {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid session"})
		return
	}

	err = stmt.Query(h.db, &user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		slog.Error("unlock: fetch user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch user"})
		return
	}

	// Hash the submitted Auth Key the same way it was hashed at signup.
	rehashedSubmitted := amnesia.HashAuthKey(submittedHash)

	// Constant-time comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare(rehashedSubmitted, user.AuthKeyHash) != 1 {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "wrong Vault Key"})
		return
	}

	// Auth Key verified — mark vault as unlocked.
	unlockTime := time.Now().UTC().Format(time.RFC3339)
	sess.VaultUnlockedAt = &unlockTime

	if sess.BAUserID != "" {
		// BA session: store vault unlock state in Redis via BA middleware.
		cookie, _ := r.Cookie(middleware.BASessionCookieName)
		if cookie != nil && h.ba != nil {
			if err := h.ba.SetVaultUnlocked(r.Context(), cookie.Value, time.Now().Add(24*time.Hour)); err != nil {
				slog.Error("unlock: set ba vault state", "error", err)
			}
		}
	} else {
		// Legacy Redis session: update in place.
		if err := h.session.Update(r.Context(), sess); err != nil {
			slog.Error("unlock: update session", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, UnlockResponse{
		WrappedDEK:        base64.StdEncoding.EncodeToString(user.WrappedDek),
		WrappedPrivateKey: base64.StdEncoding.EncodeToString(user.WrappedPrivateKey),
		PublicKey:         base64.StdEncoding.EncodeToString(user.PublicKey),
	})
}

// --- Logout ---

// @Summary		Logout
// @Description	Destroy session and clear cookie.
// @Tags			auth
// @Produce		json
// @Success		200	{object}	map[string]string
// @Security		SessionAuth
// @Router			/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess != nil {
		h.session.Delete(r.Context(), sess.ID)
	}
	middleware.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// --- Vault Setup (Better Auth flow) ---

type SetupVaultRequest struct {
	VaultKeyType      string `json:"vault_key_type"`      // "pin" or "passphrase"
	Salt              string `json:"salt"`                // base64
	AuthKeyHash       string `json:"auth_key_hash"`       // base64
	WrappedDEK        string `json:"wrapped_dek"`         // base64
	PublicKey         string `json:"public_key"`          // base64
	WrappedPrivateKey string `json:"wrapped_private_key"` // base64
}

type SetupVaultResponse struct {
	UserID             string `json:"user_id"`
	VaultSetupComplete bool   `json:"vault_setup_complete"`
}

// SetupVault creates a zEnv user row linked to the BA identity.
// Called after the user authenticates via Better Auth and chooses a Vault Key.
//
// @Summary		Setup vault
// @Description	Store client-generated crypto material and link to Better Auth identity.
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
	if sess == nil || sess.BAUserID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Better Auth session required"})
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

	// Check if vault already set up for this BA user.
	var existingID string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id FROM users WHERE better_auth_user_id = $1`, sess.BAUserID,
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
		table.Users.BetterAuthUserID,
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
		sess.BAUserID,
		now,
		now,
	)

	if _, err := insertStmt.Exec(h.db); err != nil {
		slog.Error("setup-vault: insert user", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to set up vault"})
		return
	}

	writeJSON(w, http.StatusCreated, SetupVaultResponse{
		UserID:             userID.String(),
		VaultSetupComplete: true,
	})
}

// --- Me (user state) ---

type MeResponse struct {
	BAUserID           string  `json:"ba_user_id"`
	Email              string  `json:"email"`
	VaultSetupComplete bool    `json:"vault_setup_complete"`
	VaultKeyType       string  `json:"vault_key_type,omitempty"`
	Salt               string  `json:"salt,omitempty"` // base64
	VaultUnlocked      bool    `json:"vault_unlocked"`
}

// Me returns the current user's auth state.
//
// @Summary		Get auth state
// @Description	Returns BA identity, vault setup status, and vault lock state.
// @Tags			auth
// @Produce		json
// @Success		200	{object}	MeResponse
// @Security		SessionAuth
// @Router			/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.BAUserID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Better Auth session required"})
		return
	}

	resp := MeResponse{
		BAUserID:      sess.BAUserID,
		Email:         sess.Email,
		VaultUnlocked: sess.IsVaultUnlocked(),
	}

	// Check if zEnv user exists (vault set up).
	var user model.Users
	stmt := SELECT(
		table.Users.VaultKeyType,
		table.Users.Salt,
	).FROM(table.Users).WHERE(
		table.Users.BetterAuthUserID.EQ(String(sess.BAUserID)),
	)

	if err := stmt.Query(h.db, &user); err == nil {
		resp.VaultSetupComplete = true
		resp.VaultKeyType = user.VaultKeyType
		resp.Salt = base64.StdEncoding.EncodeToString(user.Salt)
	}

	writeJSON(w, http.StatusOK, resp)
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
