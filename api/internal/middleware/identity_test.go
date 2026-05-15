package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/test_util"
)

var ts *test_util.TestServer

func TestMain(m *testing.M) {
	srv, cleanup := test_util.SetupServerForMain()
	ts = srv
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestRequireSession_ValidCookie(t *testing.T) {
	user := test_util.CreateIdentityUser(t, ts)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(test_util.SessionCookie(user.SessionToken))
	w := httptest.NewRecorder()

	identity := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	handler := identity.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := middleware.GetSession(r.Context())
		if sess == nil {
			t.Fatal("session is nil in context")
		}
		if sess.UserID != user.IdentityID {
			t.Errorf("identity ID = %q, want %q", sess.UserID, user.IdentityID)
		}
		if sess.Email != user.Email {
			t.Errorf("email = %q, want %q", sess.Email, user.Email)
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireSession_ValidBearerHeader(t *testing.T) {
	user := test_util.CreateIdentityUser(t, ts)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+user.SessionToken)
	w := httptest.NewRecorder()

	identity := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	handler := identity.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := middleware.GetSession(r.Context())
		if sess == nil {
			t.Fatal("session is nil")
		}
		if sess.UserID != user.IdentityID {
			t.Errorf("identity ID = %q, want %q", sess.UserID, user.IdentityID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireSession_ExpiredSession(t *testing.T) {
	user := test_util.CreateExpiredIdentityUser(t, ts)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(test_util.SessionCookie(user.SessionToken))
	w := httptest.NewRecorder()

	identity := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	handler := identity.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for expired session")
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireSession_MissingCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	identity := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	handler := identity.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without session")
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// Verify JSON error response.
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

func TestRequireSession_InvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(test_util.SessionCookie("not-a-real-token"))
	w := httptest.NewRecorder()

	identity := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	handler := identity.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with invalid token")
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireVaultUnlocked_Locked(t *testing.T) {
	user := test_util.CreateIdentityUser(t, ts)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(test_util.SessionCookie(user.SessionToken))
	w := httptest.NewRecorder()

	identity := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)

	// Chain: RequireSession → RequireVaultUnlocked → handler
	handler := identity.RequireSession(
		identity.RequireVaultUnlocked(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called with locked vault")
		})),
	)

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireVaultUnlocked_Unlocked(t *testing.T) {
	user := test_util.CreateIdentityUser(t, ts)

	// Mark vault as unlocked in Redis.
	identity := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	if err := identity.SetVaultUnlocked(
		test_util.Ctx(), user.SessionToken,
		test_util.FutureTime(),
	); err != nil {
		t.Fatalf("set vault unlocked: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(test_util.SessionCookie(user.SessionToken))
	w := httptest.NewRecorder()

	handler := identity.RequireSession(
		identity.RequireVaultUnlocked(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
