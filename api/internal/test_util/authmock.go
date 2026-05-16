package test_util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuthMock is an in-memory Better Auth–shaped HTTP server for API tests.
type AuthMock struct {
	Server *httptest.Server

	mu       sync.Mutex
	sessions map[string]authMockSession // lookup key → session
}

type authMockSession struct {
	UserID    string
	Email     string
	Name      string
	ExpiresAt time.Time
}

// NewAuthMock starts a mock auth HTTP server. Close with Server.Close().
func NewAuthMock() *AuthMock {
	m := &AuthMock{sessions: make(map[string]authMockSession)}
	mux := http.NewServeMux()
	mux.HandleFunc("/get-session", m.handleGetSession)
	m.Server = httptest.NewServer(mux)
	return m
}

func sessionLookupKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return strings.Split(raw, ".")[0]
}

// RegisterSession maps a cookie/Bearer token (or its prefix before ".") to a user.
func (m *AuthMock) RegisterSession(token, userID, email, name string, expiresAt time.Time) {
	key := sessionLookupKey(token)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[key] = authMockSession{
		UserID:    userID,
		Email:     email,
		Name:      name,
		ExpiresAt: expiresAt,
	}
}

func (m *AuthMock) handleGetSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw := extractAuthTokenFromRequest(r)
	key := sessionLookupKey(raw)
	m.mu.Lock()
	s, ok := m.sessions[key]
	m.mu.Unlock()
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("null"))
		return
	}
	if time.Now().UTC().After(s.ExpiresAt.UTC()) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	sid := uuid.New().String()
	resp := map[string]any{
		"session": map[string]any{
			"id":        sid,
			"expiresAt": s.ExpiresAt.UTC().Format(time.RFC3339Nano),
			"token":     raw,
			"createdAt": now,
			"updatedAt": now,
			"userId":    s.UserID,
		},
		"user": map[string]any{
			"id":            s.UserID,
			"name":          s.Name,
			"email":         s.Email,
			"emailVerified": true,
			"createdAt":     now,
			"updatedAt":     now,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func extractAuthTokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	for _, c := range r.Cookies() {
		if c.Name == "better-auth.session_token" || c.Name == "__Secure-better-auth.session_token" {
			if c.Value != "" {
				return c.Value
			}
		}
	}
	// Fall back to raw Cookie header (some clients only set the header).
	if ch := r.Header.Get("Cookie"); ch != "" {
		for _, part := range strings.Split(ch, ";") {
			part = strings.TrimSpace(part)
			for _, prefix := range []string{"better-auth.session_token=", "__Secure-better-auth.session_token="} {
				if strings.HasPrefix(part, prefix) {
					return strings.TrimPrefix(part, prefix)
				}
			}
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// AuthBaseURL returns the mock base URL (…/api/auth style path segment is not included;
// handlers are mounted at /get-session relative to server root).
func (m *AuthMock) AuthBaseURL() string {
	if m == nil || m.Server == nil {
		return ""
	}
	return strings.TrimRight(m.Server.URL, "/")
}

// FindRegisteredEmail is a test helper: returns the email for a user id if registered.
func (m *AuthMock) FindRegisteredEmail(userID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.UserID == userID {
			return s.Email
		}
	}
	return ""
}
