package handler_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	// . "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
	"github.com/Judeadeniji/zenv-sh/api/internal/test_util"
)

// setupProjectCtx creates an identity user, zenv user, an org, and unlocks the vault.
// Returns the session token, user ID, and org ID.
func setupProjectCtx(t *testing.T) (sessionToken string, userID uuid.UUID, orgID uuid.UUID) {
	t.Helper()
	identity := test_util.CreateIdentityUser(t, ts)
	zenvUser := test_util.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)

	// Create org via Jet.
	oid := uuid.New()
	now := time.Now().UTC()
	_, err := table.Organizations.INSERT(
		table.Organizations.ID,
		table.Organizations.Name,
		table.Organizations.Slug,
		table.Organizations.CreatedAt,
		table.Organizations.OwnerID,
	).VALUES(
		oid.String(),
		"TestOrg-"+uuid.New().String()[:8],
		"test-org-"+uuid.New().String()[:8],
		now,
		identity.IdentityID,
	).Exec(ts.DB)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Create membership via Jet.
	_, err = table.Members.INSERT(
		table.Members.ID,
		table.Members.OrganizationID,
		table.Members.UserID,
		table.Members.Role,
		table.Members.CreatedAt,
	).VALUES(
		uuid.New().String(),
		oid.String(),
		identity.IdentityID,
		"admin",
		now,
	).Exec(ts.DB)
	if err != nil {
		t.Fatalf("insert org member: %v", err)
	}

	// Unlock vault in Redis.
	idSession := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	if err := idSession.SetVaultUnlocked(context.Background(), identity.SessionToken, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("set vault unlocked: %v", err)
	}

	return identity.SessionToken, zenvUser.UserID, oid
}

func TestCreateProject_Success(t *testing.T) {
	sessionToken, _, orgID := setupProjectCtx(t)

	// Generate project crypto material.
	projectSalt := amnesia.GenerateSalt()
	projectDEK := amnesia.GenerateKey()
	projectKEK, _ := amnesia.DeriveKeys("project-vault-key", projectSalt, amnesia.KeyTypePassphrase)
	wrappedPDEK, pdNonce, err := amnesia.WrapKey(projectDEK, projectKEK)
	if err != nil {
		t.Fatalf("wrap project DEK: %v", err)
	}
	wrappedPDEKFull := append(pdNonce, wrappedPDEK...)

	// Wrapped project vault key (simulate wrapping with public key).
	wrappedPVK := amnesia.GenerateKey() // stand-in for wrapped key

	reqBody := jsonBody{
		"organization_id":           orgID.String(),
		"name":                      "my-project-" + uuid.New().String()[:8],
		"project_salt":              base64.StdEncoding.EncodeToString(projectSalt),
		"wrapped_project_dek":       base64.StdEncoding.EncodeToString(wrappedPDEKFull),
		"wrapped_project_vault_key": base64.StdEncoding.EncodeToString(wrappedPVK),
	}

	resp := doReqWithCookie(t, "POST", ts.URL+"/v1/projects", reqBody, sessionToken)
	assertStatus(t, resp, 201)

	var result struct {
		ID             string `json:"id"`
		OrganizationID string `json:"organization_id"`
		Name           string `json:"name"`
	}
	decodeJSON(t, resp, &result)

	if result.ID == "" {
		t.Error("id should not be empty")
	}
	if result.OrganizationID != orgID.String() {
		t.Errorf("organization_id = %q, want %q", result.OrganizationID, orgID.String())
	}
}

func TestListProjects_Success(t *testing.T) {
	identity := test_util.CreateIdentityUser(t, ts)
	test_util.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)
	orgID, _ := test_util.CreateProject(t, ts.DB, uuid.MustParse(identity.IdentityID))

	// Unlock vault.
	idSession := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	if err := idSession.SetVaultUnlocked(context.Background(), identity.SessionToken, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("set vault unlocked: %v", err)
	}

	listURL := fmt.Sprintf("%s/v1/projects?organization_id=%s", ts.URL, orgID.String())
	resp := doReqWithCookie(t, "GET", listURL, nil, identity.SessionToken)
	assertStatus(t, resp, 200)

	var result struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	decodeJSON(t, resp, &result)

	if len(result.Projects) < 1 {
		t.Error("expected at least 1 project")
	}
}

func TestGetProject_Success(t *testing.T) {
	identity := test_util.CreateIdentityUser(t, ts)
	test_util.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)
	_, projectID := test_util.CreateProject(t, ts.DB, uuid.MustParse(identity.IdentityID))

	// Unlock vault.
	idSession := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	if err := idSession.SetVaultUnlocked(context.Background(), identity.SessionToken, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("set vault unlocked: %v", err)
	}

	getURL := fmt.Sprintf("%s/v1/projects/%s", ts.URL, projectID.String())
	resp := doReqWithCookie(t, "GET", getURL, nil, identity.SessionToken)
	assertStatus(t, resp, 200)

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	decodeJSON(t, resp, &result)

	if result.ID != projectID.String() {
		t.Errorf("id = %q, want %q", result.ID, projectID.String())
	}
}

func TestGetProject_NotFound(t *testing.T) {
	identity := test_util.CreateIdentityUser(t, ts)
	test_util.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)

	// Unlock vault.
	idSession := middleware.NewIdentitySession(ts.DB, ts.Redis, ts.AuthClient)
	if err := idSession.SetVaultUnlocked(context.Background(), identity.SessionToken, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("set vault unlocked: %v", err)
	}

	getURL := fmt.Sprintf("%s/v1/projects/%s", ts.URL, uuid.New().String())
	resp := doReqWithCookie(t, "GET", getURL, nil, identity.SessionToken)
	assertStatus(t, resp, 404)
	resp.Body.Close()
}

func TestGetCrypto_Success(t *testing.T) {
	identity := test_util.CreateIdentityUser(t, ts)
	test_util.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)
	_, projectID := test_util.CreateProject(t, ts.DB, uuid.MustParse(identity.IdentityID))

	// Service token for SDK access.
	svcToken := test_util.CreateServiceToken(t, ts.DB, projectID, "development", "read")

	cryptoURL := fmt.Sprintf("%s/v1/sdk/projects/%s/crypto", ts.URL, projectID.String())
	resp := doReq(t, "GET", cryptoURL, nil, svcToken)
	assertStatus(t, resp, 200)

	var result struct {
		ProjectSalt       string `json:"project_salt"`
		WrappedProjectDEK string `json:"wrapped_project_dek"`
	}
	decodeJSON(t, resp, &result)

	if result.ProjectSalt == "" {
		t.Error("project_salt should not be empty")
	}
	if result.WrappedProjectDEK == "" {
		t.Error("wrapped_project_dek should not be empty")
	}
}
