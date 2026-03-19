package middleware

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"net/http"
	"strings"
	"time"

	. "github.com/go-jet/jet/v2/postgres"

	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

type tokenContextKey string

const (
	tokenInfoKey tokenContextKey = "token_info"
)

// TokenInfo represents the authenticated service token's scope.
type TokenInfo struct {
	TokenID     string
	ProjectID   string
	Environment string
	Permission  string // "read" or "read_write"
}

// IsWriteAllowed returns true if this token has write permission.
func (t *TokenInfo) IsWriteAllowed() bool {
	return t.Permission == "read_write"
}

// GetTokenInfo retrieves token info from the request context.
func GetTokenInfo(ctx context.Context) *TokenInfo {
	info, _ := ctx.Value(tokenInfoKey).(*TokenInfo)
	return info
}

// TokenAuth is middleware that authenticates requests via Bearer token.
// It hashes the token, looks it up in the database, checks revocation and expiry,
// and injects TokenInfo into the context.
type TokenAuth struct {
	db *sql.DB
}

func NewTokenAuth(db *sql.DB) *TokenAuth {
	return &TokenAuth{db: db}
}

// Authenticate validates the service token and injects scope into context.
func (ta *TokenAuth) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"missing or invalid Authorization header"}`, http.StatusUnauthorized)
			return
		}

		tokenPlaintext := strings.TrimPrefix(authHeader, "Bearer ")
		if !strings.HasPrefix(tokenPlaintext, "svc_") {
			http.Error(w, `{"error":"invalid token format"}`, http.StatusUnauthorized)
			return
		}

		// Hash the token and look it up.
		hash := sha256.Sum256([]byte(tokenPlaintext))

		var token model.ServiceTokens
		stmt := SELECT(
			table.ServiceTokens.ID,
			table.ServiceTokens.ProjectID,
			table.ServiceTokens.Environment,
			table.ServiceTokens.Permission,
			table.ServiceTokens.RevokedAt,
			table.ServiceTokens.ExpiresAt,
		).FROM(table.ServiceTokens).WHERE(
			table.ServiceTokens.TokenHash.EQ(Bytea(hash[:])),
		)

		if err := stmt.Query(ta.db, &token); err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Check revocation.
		if token.RevokedAt != nil {
			http.Error(w, `{"error":"token has been revoked"}`, http.StatusUnauthorized)
			return
		}

		// Check expiry.
		if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
			http.Error(w, `{"error":"token has expired"}`, http.StatusUnauthorized)
			return
		}

		info := &TokenInfo{
			TokenID:     token.ID.String(),
			ProjectID:   token.ProjectID.String(),
			Environment: token.Environment,
			Permission:  token.Permission,
		}

		ctx := context.WithValue(r.Context(), tokenInfoKey, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireWrite rejects requests from read-only tokens.
func RequireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := GetTokenInfo(r.Context())
		if info == nil || !info.IsWriteAllowed() {
			http.Error(w, `{"error":"write permission required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
