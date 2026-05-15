package middleware

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Judeadeniji/zenv-sh/api/internal/audit"
	"github.com/Judeadeniji/zenv-sh/api/internal/auth_client"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

const (
	IdentitySessionCookie       = "better-auth.session_token"
	IdentitySessionCookieSecure = "__Secure-better-auth.session_token"
	vaultUnlockPrefix           = "vault_unlock:"
)

type IdentitySession struct {
	db   *sql.DB
	rdb  *redis.Client
	auth *auth_client.Client
}

func NewIdentitySession(db *sql.DB, rdb *redis.Client, auth *auth_client.Client) *IdentitySession {
	return &IdentitySession{db: db, rdb: rdb, auth: auth}
}

// RequireSession validates the session with the auth server (GET /get-session),
// then loads vault-setup state from the identities table in Postgres.
func (id *IdentitySession) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			slog.Debug("identity: no token found in cookie or header")
			jsonError(w, "authentication required", http.StatusUnauthorized)
			return
		}

		slog.Debug("identity: resolving session", "token_prefix", token[:min(8, len(token))]+"...")

		info, err := id.auth.GetSession(r.Context(), r)
		if err != nil {
			slog.Error("identity: auth get-session", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		if info == nil || info.UserID == "" {
			slog.Debug("identity: no matching session", "token_prefix", token[:min(8, len(token))]+"...")
			jsonError(w, "session expired or invalid", http.StatusUnauthorized)
			return
		}

		authUID, err := uuid.Parse(info.UserID)
		if err != nil {
			slog.Debug("identity: invalid user id from auth", "error", err)
			jsonError(w, "session expired or invalid", http.StatusUnauthorized)
			return
		}

		var countResult struct {
			Count int64 `alias:"count"`
		}
		if err := SELECT(COUNT(table.Identities.ID).AS("count")).
			FROM(table.Identities).
			WHERE(table.Identities.IdentityID.EQ(UUID(authUID))).
			QueryContext(r.Context(), id.db, &countResult); err != nil {
			slog.Error("identity: check identities", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		hasIdentity := countResult.Count > 0

		var vaultUnlockedAt *string
		if val, redisErr := id.rdb.Get(r.Context(), vaultUnlockPrefix+token).Result(); redisErr == nil && val != "" {
			vaultUnlockedAt = &val
		}

		sess := &Session{
			ID:              info.SessionID,
			UserID:          info.UserID,
			Email:           info.Email,
			Name:            info.Name,
			VaultUnlockedAt: vaultUnlockedAt,
			HasIdentity:     hasIdentity,
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, sess)
		ctx = audit.SetUserID(ctx, authUID)
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

func extractToken(r *http.Request) string {
	if cookie, err := r.Cookie(IdentitySessionCookieSecure); err == nil && cookie.Value != "" {
		return strings.Split(cookie.Value, ".")[0]
	}
	if cookie, err := r.Cookie(IdentitySessionCookie); err == nil && cookie.Value != "" {
		return strings.Split(cookie.Value, ".")[0]
	}
	if h := r.Header.Get("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.Split(strings.TrimSpace(h[7:]), ".")[0]
	}
	return ""
}
