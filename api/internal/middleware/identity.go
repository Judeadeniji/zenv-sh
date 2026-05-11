package middleware

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Judeadeniji/zenv-sh/api/internal/audit"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

const (
	IdentitySessionCookie       = "better-auth.session_token"
	IdentitySessionCookieSecure = "__Secure-better-auth.session_token"
	vaultUnlockPrefix           = "vault_unlock:"
)

type IdentitySession struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewIdentitySession(db *sql.DB, rdb *redis.Client) *IdentitySession {
	return &IdentitySession{db: db, rdb: rdb}
}

type sessionRow struct {
	model.Sessions
	model.Users
	Identity *model.Identities
}

// RequireSession reads the identity session cookie, validates it against Postgres,
// and injects a Session into context. A single query joins sessions → users → identities
// (LEFT JOIN) so vault setup state is resolved in the same round trip.
func (id *IdentitySession) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			slog.Debug("identity: no token found in cookie or header")
			jsonError(w, "authentication required", http.StatusUnauthorized)
			return
		}

		slog.Debug("identity: resolving session", "token_prefix", token[:min(8, len(token))]+"...")

		var row sessionRow
		stmt := SELECT(
			table.Sessions.ID,
			table.Sessions.UserID,
			table.Sessions.ExpiresAt,
			table.Users.Email,
			table.Identities.ID,
		).FROM(
			table.Sessions.
				INNER_JOIN(table.Users, table.Users.ID.EQ(table.Sessions.UserID)).
				LEFT_JOIN(table.Identities, table.Identities.IdentityID.EQ(table.Users.ID)),
		).WHERE(
			table.Sessions.Token.EQ(String(token)).
				AND(table.Sessions.ExpiresAt.GT(TimestampT(time.Now().UTC()))),
		)

		if err := stmt.QueryContext(r.Context(), id.db, &row); err != nil {
			if errors.Is(err, qrm.ErrNoRows) {
				slog.Debug("identity: no matching session", "token_prefix", token[:min(8, len(token))]+"...")
				jsonError(w, "session expired or invalid", http.StatusUnauthorized)
				return
			}
			slog.Error("identity: query session", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		var vaultUnlockedAt *string
		if val, redisErr := id.rdb.Get(r.Context(), vaultUnlockPrefix+token).Result(); redisErr == nil && val != "" {
			vaultUnlockedAt = &val
		}

		sess := &Session{
			ID:              row.Sessions.ID.String(),
			UserID:          row.Sessions.UserID.String(),
			Email:           row.Users.Email,
			VaultUnlockedAt: vaultUnlockedAt,
			HasIdentity:     row.Identity != nil,
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, sess)
		if uid, err := uuid.Parse(sess.UserID); err == nil {
			ctx = audit.SetUserID(ctx, uid)
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

func extractToken(r *http.Request) string {
	if cookie, err := r.Cookie(IdentitySessionCookieSecure); err == nil && cookie.Value != "" {
		return strings.Split(cookie.Value, ".")[0]
	}
	if cookie, err := r.Cookie(IdentitySessionCookie); err == nil && cookie.Value != "" {
		return strings.Split(cookie.Value, ".")[0]
	}
	if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}
