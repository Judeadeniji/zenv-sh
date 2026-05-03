package middleware

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Judeadeniji/zenv-sh/api/internal/audit"
)

const (
	IdentitySessionCookie       = "better-auth.session_token"
	IdentitySessionCookieSecure = "__Secure-better-auth.session_token"
	vaultUnlockPrefix           = "vault_unlock:" // Redis key prefix for vault unlock state
)

// IdentitySession is middleware that reads an identity session cookie,
// verifies it against the session table in Postgres, and injects
// a zEnv Session into context. Vault unlock state is tracked in Redis.
type IdentitySession struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewIdentitySession(db *sql.DB, rdb *redis.Client) *IdentitySession {
	return &IdentitySession{db: db, rdb: rdb}
}

// identityRow holds the result of querying the identity session + user tables.
type identityRow struct {
	SessionID  string
	IdentityID string
	Email      string
	ExpiresAt  time.Time
}

// RequireSession reads the identity session cookie, validates it against Postgres,
// resolves the zEnv user, and injects a Session into context.
func (id *IdentitySession) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// 1. Try Secure cookie first (Production)
		if cookie, err := r.Cookie(IdentitySessionCookieSecure); err == nil && cookie.Value != "" {
			token = strings.Split(cookie.Value, ".")[0]
		} else if cookie, err := r.Cookie(IdentitySessionCookie); err == nil && cookie.Value != "" {
			// 2. Fall back to standard cookie (Local development)
			token = strings.Split(cookie.Value, ".")[0]
		} else if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
			// 3. Fall back to Authorization header (Postman / Cross-origin)
			token = h[7:]
		}

		if token == "" {
			slog.Debug("identity: no token found in cookie or header")
			jsonError(w, "authentication required", http.StatusUnauthorized)
			return
		}

		slog.Debug("identity: resolving session", "token_prefix", token[:min(8, len(token))]+"...")

		// Query the identity session + user tables (raw SQL — not in Go-Jet codegen).
		var row identityRow
		err := id.db.QueryRowContext(r.Context(),
			`SELECT s.id, s.user_id, u.email, s.expires_at
			 FROM "session" s
			 JOIN "user" u ON s.user_id = u.id
			 WHERE s.token = $1 AND s.expires_at > NOW()`,
			token,
		).Scan(&row.SessionID, &row.IdentityID, &row.Email, &row.ExpiresAt)
		if err != nil {
			if err == sql.ErrNoRows {
				slog.Debug("identity: no matching session", "token_prefix", token[:min(8, len(token))]+"...")
				jsonError(w, "session expired or invalid", http.StatusUnauthorized)
				return
			}
			slog.Error("identity: query session", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Resolve zEnv user by identity_id.
		var zenvUserID string
		err = id.db.QueryRowContext(r.Context(),
			`SELECT id FROM users WHERE identity_id = $1`,
			row.IdentityID,
		).Scan(&zenvUserID)
		if err != nil && err != sql.ErrNoRows {
			slog.Error("identity: resolve zenv user", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		// zenvUserID may be empty if vault hasn't been set up yet — that's OK.
		// The /auth/me and /auth/setup-vault endpoints handle that state.

		// Check vault unlock state in Redis.
		var vaultUnlockedAt *string
		val, redisErr := id.rdb.Get(r.Context(), vaultUnlockPrefix+token).Result()
		if redisErr == nil && val != "" {
			vaultUnlockedAt = &val
		}

		sess := &Session{
			ID:              row.SessionID,
			UserID:          zenvUserID, // may be empty if vault not set up
			IdentityID:      row.IdentityID,
			Email:           row.Email,
			VaultUnlockedAt: vaultUnlockedAt,
			CreatedAt:       "",
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, sess)
		if sess.UserID != "" {
			if uid, err := uuid.Parse(sess.UserID); err == nil {
				ctx = audit.SetUserID(ctx, uid)
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireVaultUnlocked checks that the user has completed both auth layers.
func (id *IdentitySession) RequireVaultUnlocked(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		if sess == nil {
			jsonError(w, "authentication required", http.StatusUnauthorized)
			return
		}
		if !sess.IsVaultUnlocked() {
			jsonError(w, "vault is locked — submit your Vault Key first", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetVaultUnlocked marks the vault as unlocked for a session token in Redis.
func (id *IdentitySession) SetVaultUnlocked(ctx context.Context, sessionToken string, expiresAt time.Time) error {
	unlockTime := time.Now().UTC().Format(time.RFC3339)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = sessionTTL
	}
	return id.rdb.Set(ctx, vaultUnlockPrefix+sessionToken, unlockTime, ttl).Err()
}

// ClearVaultUnlocked removes the vault unlock record for a session, re-locking the vault.
func (id *IdentitySession) ClearVaultUnlocked(ctx context.Context, sessionToken string) error {
	return id.rdb.Del(ctx, vaultUnlockPrefix+sessionToken).Err()
}
