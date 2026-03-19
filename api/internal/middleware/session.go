package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	sessionContextKey contextKey = "session"
	sessionCookieName            = "zenv_session"
	sessionPrefix                = "session:"
	sessionTTL                   = 24 * time.Hour
)

// Session represents an authenticated user session.
// The vault is NOT unlocked until VaultUnlockedAt is set (two-layer auth).
type Session struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`             // zEnv user UUID (may be empty if vault not set up)
	BAUserID        string  `json:"ba_user_id,omitempty"` // Better Auth user ID (set by BA middleware)
	Email           string  `json:"email"`
	VaultUnlockedAt *string `json:"vault_unlocked_at,omitempty"` // set after Vault Key verification
	CreatedAt       string  `json:"created_at"`
}

// IsVaultUnlocked returns true if the user has completed both auth layers.
func (s *Session) IsVaultUnlocked() bool {
	return s.VaultUnlockedAt != nil
}

// SessionManager handles Redis-backed sessions.
type SessionManager struct {
	rdb *redis.Client
}

func NewSessionManager(rdb *redis.Client) *SessionManager {
	return &SessionManager{rdb: rdb}
}

// Create stores a new session in Redis and returns the session ID.
func (sm *SessionManager) Create(ctx context.Context, userID, email string) (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		ID:        id,
		UserID:    userID,
		Email:     email,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(sess)
	if err != nil {
		return nil, fmt.Errorf("session: marshal: %w", err)
	}

	if err := sm.rdb.Set(ctx, sessionPrefix+id, data, sessionTTL).Err(); err != nil {
		return nil, fmt.Errorf("session: redis set: %w", err)
	}

	return sess, nil
}

// Get retrieves a session from Redis by ID.
func (sm *SessionManager) Get(ctx context.Context, id string) (*Session, error) {
	data, err := sm.rdb.Get(ctx, sessionPrefix+id).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("session: redis get: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("session: unmarshal: %w", err)
	}
	return &sess, nil
}

// Update saves a modified session back to Redis.
func (sm *SessionManager) Update(ctx context.Context, sess *Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}

	// Keep remaining TTL.
	ttl, err := sm.rdb.TTL(ctx, sessionPrefix+sess.ID).Result()
	if err != nil || ttl <= 0 {
		ttl = sessionTTL
	}

	return sm.rdb.Set(ctx, sessionPrefix+sess.ID, data, ttl).Err()
}

// Delete removes a session (logout).
func (sm *SessionManager) Delete(ctx context.Context, id string) error {
	return sm.rdb.Del(ctx, sessionPrefix+id).Err()
}

// RequireSession is middleware that loads the session from cookie.
// Rejects the request if no valid session exists (identity layer must pass first).
func (sm *SessionManager) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		sess, err := sm.Get(r.Context(), cookie.Value)
		if err != nil || sess == nil {
			http.Error(w, `{"error":"session expired or invalid"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireVaultUnlocked is middleware that requires both identity + vault key layers.
func (sm *SessionManager) RequireVaultUnlocked(next http.Handler) http.Handler {
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

// GetSession retrieves the session from request context.
func GetSession(ctx context.Context) *Session {
	sess, _ := ctx.Value(sessionContextKey).(*Session)
	return sess
}

// SetSessionCookie sets the session cookie on the response.
func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

// ClearSessionCookie removes the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}
