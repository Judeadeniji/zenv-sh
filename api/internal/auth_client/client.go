// Package auth_client is the sole bridge between zenv-api and Better Auth.
//
// There is no Go Better Auth library in this module: every call is plain HTTP
// to the apps/auth server. Route shapes and payloads follow the OpenAPI export
// at apps/auth/api-1.json (Better Auth instance API reference).
//
// User profile fields for the dashboard (name, email by user id) come from
// public.users in Postgres via internal/user_lookup — not from Better Auth admin routes.
package auth_client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a small HTTP client for the Better Auth server. It does not import
// or embed any Better Auth Go code — only net/http + JSON against AUTH_SERVER_URL.
type Client struct {
	baseURL       string
	httpClient    *http.Client
	sessionClient *http.Client // no cookies jar pollution between requests
}

// New returns a client for baseURL (e.g. https://auth.example.com/api/auth) with no trailing slash issues.
func New(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	tr := &http.Transport{Proxy: http.ProxyFromEnvironment}
	return &Client{
		baseURL:       baseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second, Transport: tr},
		sessionClient: &http.Client{Timeout: 10 * time.Second, Transport: tr},
	}
}

// SessionInfo is a normalized view of GET /get-session.
type SessionInfo struct {
	SessionID string
	UserID    string
	Email     string
	Name      string
	ExpiresAt time.Time
}

type getSessionEnvelope struct {
	Session *struct {
		ID        string `json:"id"`
		ExpiresAt string `json:"expiresAt"`
		Token     string `json:"token"`
		UserID    string `json:"userId"`
	} `json:"session"`
	User *struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
}

// GetSession validates the caller's session via the auth server (GET /get-session).
// It forwards the incoming Authorization header and/or Cookie header from r.
func (c *Client) GetSession(ctx context.Context, r *http.Request) (*SessionInfo, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("auth_client: missing base URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/get-session", nil)
	if err != nil {
		return nil, err
	}
	if h := r.Header.Get("Authorization"); h != "" {
		req.Header.Set("Authorization", h)
	}
	if cookie := cookieHeaderFromRequest(r); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := c.sessionClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth get-session: status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	bs := strings.TrimSpace(string(body))
	if bs == "" || bs == "null" {
		return nil, nil
	}

	var env getSessionEnvelope
	if err := json.Unmarshal([]byte(bs), &env); err != nil {
		return nil, fmt.Errorf("auth get-session: decode: %w", err)
	}
	if env.Session == nil || env.User == nil {
		return nil, nil
	}
	exp, err := time.Parse(time.RFC3339Nano, env.Session.ExpiresAt)
	if err != nil {
		exp, err = time.Parse(time.RFC3339, env.Session.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("auth get-session: bad expiresAt")
		}
	}
	if time.Now().UTC().After(exp.UTC()) {
		return nil, nil
	}
	return &SessionInfo{
		SessionID: env.Session.ID,
		UserID:    firstNonEmpty(env.User.ID, env.Session.UserID),
		Email:     env.User.Email,
		Name:      env.User.Name,
		ExpiresAt: exp.UTC(),
	}, nil
}

func cookieHeaderFromRequest(r *http.Request) string {
	if r.Header.Get("Cookie") != "" {
		return r.Header.Get("Cookie")
	}
	var parts []string
	for _, ck := range r.Cookies() {
		parts = append(parts, ck.Name+"="+ck.Value)
	}
	return strings.Join(parts, "; ")
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
