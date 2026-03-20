package handler_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/testutil"
)

// setupTokenCtx creates a full context for token tests:
// identity user -> zenv user -> project, then marks vault as unlocked in Redis.
// Returns the session token and project ID.
func setupTokenCtx(t *testing.T) (sessionToken string, projectID string) {
	t.Helper()
	identity := testutil.CreateIdentityUser(t, ts.DB)
	zenvUser := testutil.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)
	_, pid := testutil.CreateProject(t, ts.DB, zenvUser.UserID)

	// Mark vault as unlocked in Redis.
	idSession := middleware.NewIdentitySession(ts.DB, ts.Redis)
	err := idSession.SetVaultUnlocked(context.Background(), identity.SessionToken, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("set vault unlocked: %v", err)
	}

	return identity.SessionToken, pid.String()
}

func TestCreateToken_Success(t *testing.T) {
	sessionToken, projectID := setupTokenCtx(t)

	reqBody := jsonBody{
		"project_id":  projectID,
		"name":        "ci-deploy-token",
		"environment": "development",
		"permission":  "read",
	}

	resp := doReqWithCookie(t, "POST", ts.URL+"/v1/tokens", reqBody, sessionToken)
	assertStatus(t, resp, 201)

	var result struct {
		ID          string `json:"id"`
		Token       string `json:"token"`
		Name        string `json:"name"`
		Environment string `json:"environment"`
		Permission  string `json:"permission"`
	}
	decodeJSON(t, resp, &result)

	if result.ID == "" {
		t.Error("id should not be empty")
	}
	if !strings.HasPrefix(result.Token, "ze_") {
		t.Errorf("token should start with 'ze_', got %q", result.Token)
	}
	if result.Name != "ci-deploy-token" {
		t.Errorf("name = %q, want 'ci-deploy-token'", result.Name)
	}
	if result.Environment != "development" {
		t.Errorf("environment = %q, want 'development'", result.Environment)
	}
	if result.Permission != "read" {
		t.Errorf("permission = %q, want 'read'", result.Permission)
	}
}

func TestListTokens_Success(t *testing.T) {
	sessionToken, projectID := setupTokenCtx(t)

	// Create two tokens.
	for i := 0; i < 2; i++ {
		reqBody := jsonBody{
			"project_id":  projectID,
			"name":        fmt.Sprintf("token-%d", i),
			"environment": "development",
			"permission":  "read",
		}
		resp := doReqWithCookie(t, "POST", ts.URL+"/v1/tokens", reqBody, sessionToken)
		assertStatus(t, resp, 201)
		resp.Body.Close()
	}

	// List tokens.
	listURL := fmt.Sprintf("%s/v1/tokens?project_id=%s", ts.URL, projectID)
	resp := doReqWithCookie(t, "GET", listURL, nil, sessionToken)
	assertStatus(t, resp, 200)

	var result struct {
		Tokens []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"tokens"`
	}
	decodeJSON(t, resp, &result)

	if len(result.Tokens) < 2 {
		t.Errorf("expected at least 2 tokens, got %d", len(result.Tokens))
	}
}

func TestRevokeToken_Success(t *testing.T) {
	sessionToken, projectID := setupTokenCtx(t)

	// Create a token.
	reqBody := jsonBody{
		"project_id":  projectID,
		"name":        "revoke-me",
		"environment": "development",
		"permission":  "read",
	}
	resp := doReqWithCookie(t, "POST", ts.URL+"/v1/tokens", reqBody, sessionToken)
	assertStatus(t, resp, 201)

	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &created)

	// Revoke it.
	deleteURL := fmt.Sprintf("%s/v1/tokens/%s", ts.URL, created.ID)
	resp = doReqWithCookie(t, "DELETE", deleteURL, nil, sessionToken)
	assertStatus(t, resp, 200)
	resp.Body.Close()
}
