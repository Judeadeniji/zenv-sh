package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type contextKey string

const (
	sessionContextKey contextKey = "session"
	sessionTTL                   = 24 * time.Hour
)

// Session represents an authenticated user session.
// The vault is NOT unlocked until VaultUnlockedAt is set (two-layer auth).
// HasIdentity is false if the user has never completed vault setup.
type Session struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	Email           string  `json:"email"`
	Name            string  `json:"name,omitempty"`
	VaultUnlockedAt *string `json:"vault_unlocked_at,omitempty"`
	HasIdentity     bool    `json:"has_identity"`
}

// IsVaultUnlocked returns true if the user has completed both auth layers.
func (s *Session) IsVaultUnlocked() bool {
	return s.VaultUnlockedAt != nil
}

// GetSession retrieves the session from request context.
func GetSession(ctx context.Context) *Session {
	sess, _ := ctx.Value(sessionContextKey).(*Session)
	return sess
}

// jsonError writes a JSON error response.
func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
