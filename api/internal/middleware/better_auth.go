package middleware

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	BASessionCookieName = "better-auth.session_token"
	baVaultPrefix       = "ba_vault:" // Redis key prefix for vault unlock state
)

// BetterAuthSession is middleware that reads a Better Auth session cookie,
// verifies it against BA's session table in Postgres, and injects
// a zEnv Session into context. Vault unlock state is tracked in Redis.
type BetterAuthSession struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewBetterAuthSession(db *sql.DB, rdb *redis.Client) *BetterAuthSession {
	return &BetterAuthSession{db: db, rdb: rdb}
}

// baSessionRow holds the result of querying BA's session + user tables.
type baSessionRow struct {
	SessionID string
	BAUserID  string
	Email     string
	ExpiresAt time.Time
}

// RequireSession reads the BA session cookie, validates it against Postgres,
// resolves the zEnv user, and injects a Session into context.
func (ba *BetterAuthSession) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(BASessionCookieName)
		if err != nil {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		token := cookie.Value
		if token == "" {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		// Query BA's session + user tables (raw SQL — BA tables are not in Go-Jet codegen).
		var row baSessionRow
		err = ba.db.QueryRowContext(r.Context(),
			`SELECT s.id, s.user_id, u.email, s.expires_at
			 FROM session s
			 JOIN "user" u ON s.user_id = u.id
			 WHERE s.token = $1 AND s.expires_at > NOW()`,
			token,
		).Scan(&row.SessionID, &row.BAUserID, &row.Email, &row.ExpiresAt)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, `{"error":"session expired or invalid"}`, http.StatusUnauthorized)
				return
			}
			slog.Error("better_auth: query session", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		// Resolve zEnv user by better_auth_user_id.
		var zenvUserID string
		err = ba.db.QueryRowContext(r.Context(),
			`SELECT id FROM users WHERE better_auth_user_id = $1`,
			row.BAUserID,
		).Scan(&zenvUserID)
		if err != nil && err != sql.ErrNoRows {
			slog.Error("better_auth: resolve zenv user", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		// zenvUserID may be empty if vault hasn't been set up yet — that's OK.
		// The /auth/me and /auth/setup-vault endpoints handle that state.

		// Check vault unlock state in Redis.
		var vaultUnlockedAt *string
		val, redisErr := ba.rdb.Get(r.Context(), baVaultPrefix+token).Result()
		if redisErr == nil && val != "" {
			vaultUnlockedAt = &val
		}

		sess := &Session{
			ID:              row.SessionID,
			UserID:          zenvUserID, // may be empty if vault not set up
			BAUserID:        row.BAUserID,
			Email:           row.Email,
			VaultUnlockedAt: vaultUnlockedAt,
			CreatedAt:       "",
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireVaultUnlocked reuses the same check from session.go.
// It's defined on BetterAuthSession for middleware chaining convenience.
func (ba *BetterAuthSession) RequireVaultUnlocked(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSession(r.Context())
		if sess == nil {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		if !sess.IsVaultUnlocked() {
			http.Error(w, `{"error":"vault is locked — submit your Vault Key first"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetVaultUnlocked marks the vault as unlocked for a BA session token in Redis.
func (ba *BetterAuthSession) SetVaultUnlocked(ctx context.Context, sessionToken string, expiresAt time.Time) error {
	unlockTime := time.Now().UTC().Format(time.RFC3339)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = sessionTTL
	}
	return ba.rdb.Set(ctx, baVaultPrefix+sessionToken, unlockTime, ttl).Err()
}
