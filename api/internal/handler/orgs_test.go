package handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/testutil"
)

// setupOrgCtx creates an identity user, zenv user, and unlocks the vault.
// Returns the session token and zenv user ID.
func setupOrgCtx(t *testing.T) (sessionToken string, userID uuid.UUID) {
	t.Helper()
	identity := testutil.CreateIdentityUser(t, ts.DB)
	zenvUser := testutil.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)

	// Unlock vault in Redis.
	idSession := middleware.NewIdentitySession(ts.DB, ts.Redis)
	if err := idSession.SetVaultUnlocked(context.Background(), identity.SessionToken, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("set vault unlocked: %v", err)
	}

	return identity.SessionToken, zenvUser.UserID
}

func TestCreateOrg_Success(t *testing.T) {
	sessionToken, _ := setupOrgCtx(t)

	reqBody := jsonBody{
		"name": "acme-corp-" + uuid.New().String()[:8],
	}

	resp := doReqWithCookie(t, "POST", ts.URL+"/v1/orgs", reqBody, sessionToken)
	assertStatus(t, resp, 201)

	var result struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		OwnerID string `json:"owner_id"`
	}
	decodeJSON(t, resp, &result)

	if result.ID == "" {
		t.Error("id should not be empty")
	}
	if result.OwnerID == "" {
		t.Error("owner_id should not be empty")
	}
}

func TestListOrgs_Success(t *testing.T) {
	sessionToken, _ := setupOrgCtx(t)

	// Create an org first.
	reqBody := jsonBody{
		"name": "list-org-" + uuid.New().String()[:8],
	}
	resp := doReqWithCookie(t, "POST", ts.URL+"/v1/orgs", reqBody, sessionToken)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	// List orgs.
	resp = doReqWithCookie(t, "GET", ts.URL+"/v1/orgs", nil, sessionToken)
	assertStatus(t, resp, 200)

	var result struct {
		Organizations []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"organizations"`
	}
	decodeJSON(t, resp, &result)

	if len(result.Organizations) < 1 {
		t.Error("expected at least 1 organization")
	}
}

func TestGetOrg_Success(t *testing.T) {
	sessionToken, userID := setupOrgCtx(t)

	// Create org via fixture.
	orgID, _ := testutil.CreateProject(t, ts.DB, userID) // CreateProject also creates an org

	getURL := fmt.Sprintf("%s/v1/orgs/%s", ts.URL, orgID.String())
	resp := doReqWithCookie(t, "GET", getURL, nil, sessionToken)
	assertStatus(t, resp, 200)

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	decodeJSON(t, resp, &result)

	if result.ID != orgID.String() {
		t.Errorf("id = %q, want %q", result.ID, orgID.String())
	}
}

func TestGetOrg_NotFound(t *testing.T) {
	sessionToken, _ := setupOrgCtx(t)

	getURL := fmt.Sprintf("%s/v1/orgs/%s", ts.URL, uuid.New().String())
	resp := doReqWithCookie(t, "GET", getURL, nil, sessionToken)
	assertStatus(t, resp, 404)
	resp.Body.Close()
}

func TestListMembers_Success(t *testing.T) {
	sessionToken, userID := setupOrgCtx(t)

	// CreateProject creates an org with the user as admin member.
	orgID, _ := testutil.CreateProject(t, ts.DB, userID)

	membersURL := fmt.Sprintf("%s/v1/orgs/%s/members", ts.URL, orgID.String())
	resp := doReqWithCookie(t, "GET", membersURL, nil, sessionToken)
	assertStatus(t, resp, 200)

	var result struct {
		Members []struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"members"`
	}
	decodeJSON(t, resp, &result)

	if len(result.Members) < 1 {
		t.Fatal("expected at least 1 member")
	}

	found := false
	for _, m := range result.Members {
		if m.UserID == userID.String() && m.Role == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected owner to be an admin member")
	}
}

func TestAddMember_Success(t *testing.T) {
	sessionToken, userID := setupOrgCtx(t)
	orgID, _ := testutil.CreateProject(t, ts.DB, userID)

	// Create a second zenv user to add as member.
	identity2 := testutil.CreateIdentityUser(t, ts.DB)
	zenvUser2 := testutil.CreateZenvUser(t, ts.DB, identity2.IdentityID, identity2.Email)

	reqBody := jsonBody{
		"user_id": zenvUser2.UserID.String(),
		"role":    "dev",
	}

	addURL := fmt.Sprintf("%s/v1/orgs/%s/members", ts.URL, orgID.String())
	resp := doReqWithCookie(t, "POST", addURL, reqBody, sessionToken)
	assertStatus(t, resp, 201)

	var result struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	decodeJSON(t, resp, &result)

	if result.UserID != zenvUser2.UserID.String() {
		t.Errorf("user_id = %q, want %q", result.UserID, zenvUser2.UserID.String())
	}
	if result.Role != "dev" {
		t.Errorf("role = %q, want 'dev'", result.Role)
	}
}

func TestRemoveMember_Success(t *testing.T) {
	sessionToken, userID := setupOrgCtx(t)
	orgID, _ := testutil.CreateProject(t, ts.DB, userID)

	// Create and add a second user.
	identity2 := testutil.CreateIdentityUser(t, ts.DB)
	zenvUser2 := testutil.CreateZenvUser(t, ts.DB, identity2.IdentityID, identity2.Email)

	addBody := jsonBody{
		"user_id": zenvUser2.UserID.String(),
		"role":    "dev",
	}
	addURL := fmt.Sprintf("%s/v1/orgs/%s/members", ts.URL, orgID.String())
	resp := doReqWithCookie(t, "POST", addURL, addBody, sessionToken)
	assertStatus(t, resp, 201)

	var added struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &added)

	// Remove the second user.
	removeURL := fmt.Sprintf("%s/v1/orgs/%s/members/%s", ts.URL, orgID.String(), added.ID)
	resp = doReqWithCookie(t, "DELETE", removeURL, nil, sessionToken)
	assertStatus(t, resp, 200)
	resp.Body.Close()
}
