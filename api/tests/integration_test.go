package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/api/internal/testutil"
)

var ts *testutil.TestServer

func TestMain(m *testing.M) {
	srv, cleanup := testutil.SetupServerForMain()
	ts = srv
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestE2E_FullSecretLifecycle(t *testing.T) {
	// 1. Create identity user (simulates auth server signup).
	user := testutil.CreateIdentityUser(t, ts.DB)

	// 2. GET /v1/auth/me — vault not set up yet.
	resp := doReq(t, "GET", ts.URL+"/v1/auth/me", nil, withBearer(user.SessionToken))
	assertStatus(t, resp, 200)
	me := decodeJSON(t, resp)
	if me["vault_setup_complete"] != false {
		t.Fatalf("expected vault_setup_complete=false, got %v", me["vault_setup_complete"])
	}

	// 3. POST /v1/auth/setup-vault — store crypto material.
	salt := amnesia.GenerateSalt()
	kek, authKey := amnesia.DeriveKeys("e2e-vault-key", salt, amnesia.KeyTypePassphrase)
	authKeyHash := amnesia.HashAuthKey(authKey)
	dek := amnesia.GenerateKey()
	wrappedDEK, dekNonce, _ := amnesia.WrapKey(dek, kek)
	pubKey, privKey, _ := amnesia.GenerateKeypair()
	wrappedPriv, privNonce, _ := amnesia.Encrypt(privKey, dek)

	vaultBody := map[string]string{
		"vault_key_type":      "passphrase",
		"salt":                base64.StdEncoding.EncodeToString(salt),
		"auth_key_hash":       base64.StdEncoding.EncodeToString(authKeyHash),
		"wrapped_dek":         base64.StdEncoding.EncodeToString(append(dekNonce, wrappedDEK...)),
		"public_key":          base64.StdEncoding.EncodeToString(pubKey),
		"wrapped_private_key": base64.StdEncoding.EncodeToString(append(privNonce, wrappedPriv...)),
	}
	resp = doReq(t, "POST", ts.URL+"/v1/auth/setup-vault", jsonBody(t, vaultBody), withBearer(user.SessionToken))
	assertStatus(t, resp, 201)
	setupResp := decodeJSON(t, resp)
	if setupResp["vault_setup_complete"] != true {
		t.Fatalf("expected vault_setup_complete=true")
	}

	// 4. GET /v1/auth/me — vault now set up.
	resp = doReq(t, "GET", ts.URL+"/v1/auth/me", nil, withBearer(user.SessionToken))
	assertStatus(t, resp, 200)
	me = decodeJSON(t, resp)
	if me["vault_setup_complete"] != true {
		t.Fatalf("expected vault_setup_complete=true after setup")
	}

	// 5. POST /v1/auth/setup-vault — duplicate should 409.
	resp = doReq(t, "POST", ts.URL+"/v1/auth/setup-vault", jsonBody(t, vaultBody), withBearer(user.SessionToken))
	assertStatus(t, resp, 409)

	// 6. POST /v1/auth/unlock — correct vault key.
	unlockBody := map[string]string{
		"auth_key_hash": base64.StdEncoding.EncodeToString(authKey),
	}
	resp = doReq(t, "POST", ts.URL+"/v1/auth/unlock",
		jsonBody(t, unlockBody),
		withBearer(user.SessionToken), withCookie(user.SessionToken),
	)
	assertStatus(t, resp, 200)
	unlockResp := decodeJSON(t, resp)
	if unlockResp["wrapped_dek"] == nil {
		t.Fatal("unlock should return wrapped_dek")
	}

	// 7. POST /v1/auth/unlock — wrong vault key.
	wrongBody := map[string]string{
		"auth_key_hash": base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
	}
	resp = doReq(t, "POST", ts.URL+"/v1/auth/unlock",
		jsonBody(t, wrongBody),
		withBearer(user.SessionToken), withCookie(user.SessionToken),
	)
	assertStatus(t, resp, 403)

	// --- SDK flow: create project + token + secrets CRUD ---

	// 8. Create project infrastructure (via DB — would normally be dashboard).
	zu := testutil.CreateZenvUser(t, ts.DB,
		"id-sdk-"+fmt.Sprintf("%d", os.Getpid()), // different identity for SDK
		fmt.Sprintf("sdk-%d@test.zenv.sh", os.Getpid()),
	)
	_, projectID := testutil.CreateProject(t, ts.DB, zu.UserID)
	svcToken := testutil.CreateServiceToken(t, ts.DB, projectID, "development", "read_write")

	// 9. POST /v1/sdk/secrets — create.
	nameHash := base64.StdEncoding.EncodeToString(amnesia.GenerateKey())
	ciphertext := base64.StdEncoding.EncodeToString(amnesia.GenerateKey())
	nonce := base64.StdEncoding.EncodeToString(amnesia.GenerateNonce())

	secretBody := map[string]string{
		"project_id":  projectID.String(),
		"environment": "development",
		"name_hash":   nameHash,
		"ciphertext":  ciphertext,
		"nonce":       nonce,
	}
	resp = doReq(t, "POST", ts.URL+"/v1/sdk/secrets", jsonBody(t, secretBody), withBearer(svcToken))
	assertStatus(t, resp, 201)

	// 10. POST /v1/sdk/secrets — duplicate should 409.
	resp = doReq(t, "POST", ts.URL+"/v1/sdk/secrets", jsonBody(t, secretBody), withBearer(svcToken))
	assertStatus(t, resp, 409)

	// 11. GET /v1/sdk/secrets — list.
	resp = doReq(t, "GET",
		fmt.Sprintf("%s/v1/sdk/secrets?project_id=%s&environment=development", ts.URL, projectID),
		nil, withBearer(svcToken))
	assertStatus(t, resp, 200)
	listResp := decodeJSON(t, resp)
	secrets, ok := listResp["secrets"].([]interface{})
	if !ok || len(secrets) == 0 {
		t.Fatal("expected at least one secret in list")
	}

	// 12. PUT /v1/sdk/secrets/:nameHash — update.
	nhURL := base64.URLEncoding.EncodeToString(amnesia.GenerateKey()) // reuse original
	// Actually we need the URL-safe version of the original nameHash
	nhBytes, _ := base64.StdEncoding.DecodeString(nameHash)
	nhURL = base64.URLEncoding.EncodeToString(nhBytes)

	newCT := base64.StdEncoding.EncodeToString(amnesia.GenerateKey())
	newNonce := base64.StdEncoding.EncodeToString(amnesia.GenerateNonce())
	updateBody := map[string]string{
		"ciphertext": newCT,
		"nonce":      newNonce,
	}
	resp = doReq(t, "PUT",
		fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nhURL, projectID),
		jsonBody(t, updateBody), withBearer(svcToken))
	assertStatus(t, resp, 200)
	updateResp := decodeJSON(t, resp)
	if updateResp["version"] != float64(2) {
		t.Fatalf("expected version=2, got %v", updateResp["version"])
	}

	// 13. DELETE /v1/sdk/secrets/:nameHash.
	resp = doReq(t, "DELETE",
		fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nhURL, projectID),
		nil, withBearer(svcToken))
	assertStatus(t, resp, 200)

	// 14. GET deleted — should 404.
	resp = doReq(t, "GET",
		fmt.Sprintf("%s/v1/sdk/secrets/%s?project_id=%s&environment=development", ts.URL, nhURL, projectID),
		nil, withBearer(svcToken))
	assertStatus(t, resp, 404)
}

func TestE2E_ReadOnlyTokenCannotWrite(t *testing.T) {
	user := testutil.CreateIdentityUser(t, ts.DB)
	zu := testutil.CreateZenvUser(t, ts.DB, user.IdentityID, user.Email)
	_, projectID := testutil.CreateProject(t, ts.DB, zu.UserID)
	roToken := testutil.CreateServiceToken(t, ts.DB, projectID, "development", "read")

	// Read should work.
	resp := doReq(t, "GET",
		fmt.Sprintf("%s/v1/sdk/secrets?project_id=%s&environment=development", ts.URL, projectID),
		nil, withBearer(roToken))
	assertStatus(t, resp, 200)

	// Write should be forbidden.
	body := map[string]string{
		"project_id":  projectID.String(),
		"environment": "development",
		"name_hash":   base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"ciphertext":  base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"nonce":       base64.StdEncoding.EncodeToString(amnesia.GenerateNonce()),
	}
	resp = doReq(t, "POST", ts.URL+"/v1/sdk/secrets", jsonBody(t, body), withBearer(roToken))
	assertStatus(t, resp, 403)
}

func TestE2E_InvalidTokenRejected(t *testing.T) {
	resp := doReq(t, "GET",
		ts.URL+"/v1/sdk/secrets?project_id=00000000-0000-0000-0000-000000000000&environment=dev",
		nil, withBearer("ze_dev_notreal"))
	assertStatus(t, resp, 401)
}

// --- Helpers ---

type reqOption func(*http.Request)

func withBearer(token string) reqOption {
	return func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	}
}

func withCookie(token string) reqOption {
	return func(r *http.Request) {
		r.AddCookie(testutil.SessionCookie(token))
	}
}

func jsonBody(t *testing.T, v interface{}) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return bytes.NewReader(b)
}

func doReq(t *testing.T, method, url string, body io.Reader, opts ...reqOption) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, opt := range opts {
		opt(req)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, want, string(body))
	}
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return m
}
