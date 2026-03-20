package handler_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/api/internal/testutil"
)

// setupSecretCtx creates the full context for secret tests:
// identity user -> zenv user -> project -> service token.
// Returns the service token (plaintext) and project ID.
func setupSecretCtx(t *testing.T, env, permission string) (token string, projectID string) {
	t.Helper()
	identity := testutil.CreateIdentityUser(t, ts.DB)
	zenvUser := testutil.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)
	_, pid := testutil.CreateProject(t, ts.DB, zenvUser.UserID)
	svcToken := testutil.CreateServiceToken(t, ts.DB, pid, env, permission)
	return svcToken, pid.String()
}

// makeSecretBody creates a valid create-secret request body with random crypto material.
func makeSecretBody(t *testing.T, projectID, env string) jsonBody {
	t.Helper()
	nameHash := amnesia.GenerateKey() // 32 random bytes as HMAC stand-in
	ciphertext := amnesia.GenerateKey()
	nonce := amnesia.GenerateNonce()

	return jsonBody{
		"project_id":  projectID,
		"environment": env,
		"name_hash":   base64.StdEncoding.EncodeToString(nameHash),
		"ciphertext":  base64.StdEncoding.EncodeToString(ciphertext),
		"nonce":       base64.StdEncoding.EncodeToString(nonce),
	}
}

// nameHashToURL converts a StdEncoding base64 name hash to a URLEncoding base64 string for use in paths.
func nameHashToURL(t *testing.T, stdB64 string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(stdB64)
	if err != nil {
		t.Fatalf("decode name_hash: %v", err)
	}
	return base64.URLEncoding.EncodeToString(raw)
}

func TestCreateSecret_Success(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")
	body := makeSecretBody(t, projectID, "development")

	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 201)

	var result struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	decodeJSON(t, resp, &result)

	if result.ID == "" {
		t.Error("id should not be empty")
	}
	if result.Version != 1 {
		t.Errorf("version = %d, want 1", result.Version)
	}
}

func TestCreateSecret_Duplicate(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")
	body := makeSecretBody(t, projectID, "development")

	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	// Same name_hash again.
	resp = doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 409)
	resp.Body.Close()
}

func TestCreateSecret_MissingFields(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")

	// Omit ciphertext.
	body := jsonBody{
		"project_id":  projectID,
		"environment": "development",
		"name_hash":   base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"nonce":       base64.StdEncoding.EncodeToString(amnesia.GenerateNonce()),
	}

	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 400)
	resp.Body.Close()
}

func TestGetSecret_Success(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")
	body := makeSecretBody(t, projectID, "development")

	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	nameHashURL := nameHashToURL(t, body["name_hash"].(string))
	getURL := fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nameHashURL, projectID)

	resp = doReq(t, "GET", getURL, nil, token)
	assertStatus(t, resp, 200)

	var result struct {
		NameHash   string `json:"name_hash"`
		Ciphertext string `json:"ciphertext"`
		Version    int    `json:"version"`
	}
	decodeJSON(t, resp, &result)

	if result.NameHash != body["name_hash"].(string) {
		t.Errorf("name_hash mismatch: got %q, want %q", result.NameHash, body["name_hash"].(string))
	}
	if result.Ciphertext != body["ciphertext"].(string) {
		t.Errorf("ciphertext mismatch")
	}
	if result.Version != 1 {
		t.Errorf("version = %d, want 1", result.Version)
	}
}

func TestGetSecret_NotFound(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")

	nameHash := base64.URLEncoding.EncodeToString(amnesia.GenerateKey())
	getURL := fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nameHash, projectID)

	resp := doReq(t, "GET", getURL, nil, token)
	assertStatus(t, resp, 404)
	resp.Body.Close()
}

func TestListSecrets_Empty(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")

	listURL := fmt.Sprintf("%s/v1/sdk/secrets?project_id=%s&environment=development", ts.URL, projectID)
	resp := doReq(t, "GET", listURL, nil, token)
	assertStatus(t, resp, 200)

	var result struct {
		Secrets []interface{} `json:"secrets"`
	}
	decodeJSON(t, resp, &result)

	if len(result.Secrets) != 0 {
		t.Errorf("expected empty secrets array, got %d items", len(result.Secrets))
	}
}

func TestListSecrets_FiltersEnvironment(t *testing.T) {
	// Create two tokens, one for dev and one for prod, both in the same project.
	identity := testutil.CreateIdentityUser(t, ts.DB)
	zenvUser := testutil.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)
	_, pid := testutil.CreateProject(t, ts.DB, zenvUser.UserID)
	devToken := testutil.CreateServiceToken(t, ts.DB, pid, "development", "read_write")
	prodToken := testutil.CreateServiceToken(t, ts.DB, pid, "production", "read_write")
	projectID := pid.String()

	// Create a secret in dev.
	devBody := makeSecretBody(t, projectID, "development")
	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", devBody, devToken)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	// Create a secret in prod.
	prodBody := makeSecretBody(t, projectID, "production")
	resp = doReq(t, "POST", ts.URL+"/v1/sdk/secrets", prodBody, prodToken)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	// List dev only.
	listURL := fmt.Sprintf("%s/v1/sdk/secrets?project_id=%s&environment=development", ts.URL, projectID)
	resp = doReq(t, "GET", listURL, nil, devToken)
	assertStatus(t, resp, 200)

	var result struct {
		Secrets []struct {
			Environment string `json:"environment"`
		} `json:"secrets"`
	}
	decodeJSON(t, resp, &result)

	if len(result.Secrets) != 1 {
		t.Fatalf("expected 1 dev secret, got %d", len(result.Secrets))
	}
	if result.Secrets[0].Environment != "development" {
		t.Errorf("environment = %q, want development", result.Secrets[0].Environment)
	}
}

func TestUpdateSecret_Success(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")
	body := makeSecretBody(t, projectID, "development")

	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	nameHashURL := nameHashToURL(t, body["name_hash"].(string))
	updateURL := fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nameHashURL, projectID)

	updateBody := jsonBody{
		"ciphertext": base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"nonce":      base64.StdEncoding.EncodeToString(amnesia.GenerateNonce()),
	}

	resp = doReq(t, "PUT", updateURL, updateBody, token)
	assertStatus(t, resp, 200)

	var result struct {
		Version int `json:"version"`
	}
	decodeJSON(t, resp, &result)

	if result.Version != 2 {
		t.Errorf("version = %d, want 2", result.Version)
	}
}

func TestDeleteSecret_Success(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")
	body := makeSecretBody(t, projectID, "development")

	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	nameHashURL := nameHashToURL(t, body["name_hash"].(string))
	deleteURL := fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nameHashURL, projectID)

	resp = doReq(t, "DELETE", deleteURL, nil, token)
	assertStatus(t, resp, 200)
	resp.Body.Close()

	// Verify it is gone.
	getURL := fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nameHashURL, projectID)
	resp = doReq(t, "GET", getURL, nil, token)
	assertStatus(t, resp, 404)
	resp.Body.Close()
}

func TestBulkFetch_Success(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")

	body1 := makeSecretBody(t, projectID, "development")
	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body1, token)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	body2 := makeSecretBody(t, projectID, "development")
	resp = doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body2, token)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	bulkBody := jsonBody{
		"project_id":  projectID,
		"environment": "development",
		"name_hashes": []string{body1["name_hash"].(string), body2["name_hash"].(string)},
	}

	resp = doReq(t, "POST", ts.URL+"/v1/sdk/secrets/bulk", bulkBody, token)
	assertStatus(t, resp, 200)

	var result struct {
		Secrets []struct {
			NameHash string `json:"name_hash"`
		} `json:"secrets"`
	}
	decodeJSON(t, resp, &result)

	if len(result.Secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(result.Secrets))
	}
}

func TestVersions_AfterUpdate(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")
	body := makeSecretBody(t, projectID, "development")

	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	// Update to create version history.
	nameHashURL := nameHashToURL(t, body["name_hash"].(string))
	updateURL := fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nameHashURL, projectID)

	updateBody := jsonBody{
		"ciphertext": base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"nonce":      base64.StdEncoding.EncodeToString(amnesia.GenerateNonce()),
	}
	resp = doReq(t, "PUT", updateURL, updateBody, token)
	assertStatus(t, resp, 200)
	resp.Body.Close()

	// Get versions.
	versionsURL := fmt.Sprintf("%s/v1/sdk/secrets/%s/versions?project_id=%s&environment=development", ts.URL, nameHashURL, projectID)
	resp = doReq(t, "GET", versionsURL, nil, token)
	assertStatus(t, resp, 200)

	var result struct {
		Current  int `json:"current_version"`
		Versions []struct {
			Version int `json:"version"`
		} `json:"versions"`
	}
	decodeJSON(t, resp, &result)

	if result.Current != 2 {
		t.Errorf("current_version = %d, want 2", result.Current)
	}
	if len(result.Versions) != 1 {
		t.Fatalf("expected 1 archived version, got %d", len(result.Versions))
	}
	if result.Versions[0].Version != 1 {
		t.Errorf("archived version = %d, want 1", result.Versions[0].Version)
	}
}

func TestRollback_Success(t *testing.T) {
	token, projectID := setupSecretCtx(t, "development", "read_write")
	body := makeSecretBody(t, projectID, "development")
	originalCiphertext := body["ciphertext"].(string)

	resp := doReq(t, "POST", ts.URL+"/v1/sdk/secrets", body, token)
	assertStatus(t, resp, 201)
	resp.Body.Close()

	// Update.
	nameHashURL := nameHashToURL(t, body["name_hash"].(string))
	updateURL := fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nameHashURL, projectID)

	updateBody := jsonBody{
		"ciphertext": base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"nonce":      base64.StdEncoding.EncodeToString(amnesia.GenerateNonce()),
	}
	resp = doReq(t, "PUT", updateURL, updateBody, token)
	assertStatus(t, resp, 200)
	resp.Body.Close()

	// Rollback to version 1.
	rollbackURL := fmt.Sprintf("%s/v1/sdk/secrets/%s/rollback?project_id=%s&environment=development", ts.URL, nameHashURL, projectID)
	rollbackBody := jsonBody{"version": 1}

	resp = doReq(t, "POST", rollbackURL, rollbackBody, token)
	assertStatus(t, resp, 200)

	var result struct {
		Version    int    `json:"version"`
		Ciphertext string `json:"ciphertext"`
	}
	decodeJSON(t, resp, &result)

	if result.Version != 3 {
		t.Errorf("version after rollback = %d, want 3", result.Version)
	}
	if result.Ciphertext != originalCiphertext {
		t.Error("ciphertext should match original after rollback")
	}
}
