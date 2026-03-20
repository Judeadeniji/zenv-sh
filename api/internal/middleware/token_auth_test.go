package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/testutil"
)

func TestTokenAuth_ValidToken(t *testing.T) {
	user := testutil.CreateIdentityUser(t, ts.DB)
	zu := testutil.CreateZenvUser(t, ts.DB, user.IdentityID, user.Email)
	_, projectID := testutil.CreateProject(t, ts.DB, zu.UserID)
	token := testutil.CreateServiceToken(t, ts.DB, projectID, "development", "read_write")

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	ta := middleware.NewTokenAuth(ts.DB)
	handler := ta.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := middleware.GetTokenInfo(r.Context())
		if info == nil {
			t.Fatal("token info is nil")
		}
		if info.ProjectID != projectID.String() {
			t.Errorf("project ID = %q, want %q", info.ProjectID, projectID.String())
		}
		if info.Permission != "read_write" {
			t.Errorf("permission = %q, want read_write", info.Permission)
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestTokenAuth_InvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ze_dev_notarealtoken")
	w := httptest.NewRecorder()

	ta := middleware.NewTokenAuth(ts.DB)
	handler := ta.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with invalid token")
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestTokenAuth_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	ta := middleware.NewTokenAuth(ts.DB)
	handler := ta.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without auth header")
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected JSON error response")
	}
}

func TestTokenAuth_NonZePrefix(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer svc_dev_oldprefix")
	w := httptest.NewRecorder()

	ta := middleware.NewTokenAuth(ts.DB)
	handler := ta.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with wrong prefix")
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireWrite_ReadOnlyToken(t *testing.T) {
	user := testutil.CreateIdentityUser(t, ts.DB)
	zu := testutil.CreateZenvUser(t, ts.DB, user.IdentityID, user.Email)
	_, projectID := testutil.CreateProject(t, ts.DB, zu.UserID)
	token := testutil.CreateServiceToken(t, ts.DB, projectID, "development", "read")

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	ta := middleware.NewTokenAuth(ts.DB)
	handler := ta.Authenticate(
		middleware.RequireWrite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called for read-only token on write endpoint")
		})),
	)

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireWrite_ReadWriteToken(t *testing.T) {
	user := testutil.CreateIdentityUser(t, ts.DB)
	zu := testutil.CreateZenvUser(t, ts.DB, user.IdentityID, user.Email)
	_, projectID := testutil.CreateProject(t, ts.DB, zu.UserID)
	token := testutil.CreateServiceToken(t, ts.DB, projectID, "development", "read_write")

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	ta := middleware.NewTokenAuth(ts.DB)
	handler := ta.Authenticate(
		middleware.RequireWrite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
