package middleware

import (
	"context"
	"time"
)

type contextKey string

const (
	sessionContextKey contextKey = "session"
	sessionTTL                   = 24 * time.Hour
)

// Session represents an authenticated user session.
// The vault is NOT unlocked until VaultUnlockedAt is set (two-layer auth).
type Session struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`               // zEnv user UUID (may be empty if vault not set up)
	IdentityID      string  `json:"identity_id,omitempty"` // External identity provider user ID
	Email           string  `json:"email"`
	VaultUnlockedAt *string `json:"vault_unlocked_at,omitempty"` // set after Vault Key verification
	CreatedAt       string  `json:"created_at"`
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
