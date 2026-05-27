package zenv

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/sdk-go/crypto"
)

func setupMockServer(t *testing.T, vaultKey string, dek []byte, env, secretName, secretValue string) *httptest.Server {
	salt := amnesia.GenerateSalt()
	kek, _ := amnesia.DeriveKeys([]byte(vaultKey), salt, amnesia.KeyTypePassphrase)
	wrappedDEK, nonce, _ := amnesia.WrapKey(dek, kek)
	
	fullWrappedDEK := append(nonce, wrappedDEK...)

	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":       base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": base64.StdEncoding.EncodeToString(fullWrappedDEK),
		}
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		nameHash := crypto.ComputeNameHash(secretName, dek) // hmacKey is dek
		resp := map[string]interface{}{
			"secrets": []map[string]string{
				{"id": "sec_1", "name_hash": nameHash},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		nameHash := crypto.ComputeNameHash(secretName, dek)
		ct, nc, nh, _ := crypto.EncryptSecret(secretName, secretValue, dek, dek)
		
		if nh != nameHash {
			t.Errorf("Name hashes don't match")
		}

		resp := map[string]interface{}{
			"secrets": []map[string]string{
				{"id": "sec_1", "ciphertext": ct, "nonce": nc, "name_hash": nh},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestZenvClient(t *testing.T) {
	vaultKey := "test-vault-key"
	dek := amnesia.GenerateKey()
	env := "production"
	name := "API_KEY"
	val := "super-secret"

	server := setupMockServer(t, vaultKey, dek, env, name, val)
	defer server.Close()

	client, err := NewClient(
		WithAPIURL(server.URL),
		WithToken("token"),
		WithProjectID("prj_1"),
		WithVaultKey(vaultKey),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	fetchedVal, err := client.FetchSecret(env, name)
	if err != nil {
		t.Fatalf("FetchSecret failed: %v", err)
	}

	if fetchedVal != val {
		t.Errorf("Expected %s, got %s", val, fetchedVal)
	}

	all, err := client.FetchAllSecrets(env)
	if err != nil {
		t.Fatalf("FetchAllSecrets failed: %v", err)
	}

	if len(all) != 1 || all[name] != val {
		t.Errorf("FetchAllSecrets returned incorrect map: %v", all)
	}
}

func TestNewClient_BadCrypto(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		// Return invalid base64 for project_salt to force an error.
		resp := map[string]string{
			"project_salt":        "not-valid-base64!!!",
			"wrapped_project_dek": "dW5pY29ybg==",
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := NewClient(
		WithAPIURL(server.URL),
		WithToken("token"),
		WithProjectID("prj_1"),
		WithVaultKey("vault-key"),
	)
	if err == nil {
		t.Error("Expected NewClient to fail with invalid base64 salt")
	}
}

func TestNewClient_InvalidWrappedDEK(t *testing.T) {
	salt := amnesia.GenerateSalt()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		// Valid salt but invalid (too short) wrapped DEK.
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": base64.StdEncoding.EncodeToString([]byte("short")),
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := NewClient(
		WithAPIURL(server.URL),
		WithToken("token"),
		WithProjectID("prj_1"),
		WithVaultKey("vault-key"),
	)
	if err == nil {
		t.Error("Expected NewClient to fail when wrapped DEK is too short")
	}
}

func TestNewClient_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := NewClient(
		WithAPIURL(server.URL),
		WithToken("bad-token"),
		WithProjectID("prj_1"),
		WithVaultKey("vault-key"),
	)
	if err == nil {
		t.Error("Expected NewClient to fail with unauthorized API response")
	}
}

func TestFetchSecret_NotFound(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	salt := amnesia.GenerateSalt()
	kek, _ := amnesia.DeriveKeys([]byte(vaultKey), salt, amnesia.KeyTypePassphrase)
	wrappedDEK, nonce, _ := amnesia.WrapKey(dek, kek)
	fullWrappedDEK := append(nonce, wrappedDEK...)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": base64.StdEncoding.EncodeToString(fullWrappedDEK),
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		// Return empty secrets list.
		resp := map[string]interface{}{
			"secrets": []interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(
		WithAPIURL(server.URL),
		WithToken("token"),
		WithProjectID("prj_1"),
		WithVaultKey(vaultKey),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = c.FetchSecret("production", "MISSING_KEY")
	if err == nil {
		t.Error("Expected FetchSecret to return error when secret not found")
	}
}

func TestFetchAllSecrets_Empty(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	salt := amnesia.GenerateSalt()
	kek, _ := amnesia.DeriveKeys([]byte(vaultKey), salt, amnesia.KeyTypePassphrase)
	wrappedDEK, nonce, _ := amnesia.WrapKey(dek, kek)
	fullWrappedDEK := append(nonce, wrappedDEK...)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": base64.StdEncoding.EncodeToString(fullWrappedDEK),
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"secrets": []interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(
		WithAPIURL(server.URL),
		WithToken("token"),
		WithProjectID("prj_1"),
		WithVaultKey(vaultKey),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	all, err := c.FetchAllSecrets("production")
	if err != nil {
		t.Fatalf("FetchAllSecrets failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("Expected empty map, got %v", all)
	}
}

func TestNewClient_EnvFallback(t *testing.T) {
	vaultKey := "correct-horse-battery-staple"
	dek := []byte("12345678901234567890123456789012")
	env := "dev"
	name := "MY_SECRET"
	val := "env-fallback-value"

	server := setupMockServer(t, vaultKey, dek, env, name, val)
	defer server.Close()

	t.Setenv("ZENV_API_URL", server.URL)
	t.Setenv("ZENV_TOKEN", "token")
	t.Setenv("ZENV_PROJECT", "prj_1")
	t.Setenv("ZENV_PROJECT_KEY", vaultKey)

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient failed with env fallback: %v", err)
	}

	fetched, err := client.FetchSecret(env, name)
	if err != nil {
		t.Fatalf("FetchSecret failed: %v", err)
	}
	if fetched != val {
		t.Errorf("Expected %s, got %s", val, fetched)
	}
}

func TestClientZero(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	salt := amnesia.GenerateSalt()
	kek, _ := amnesia.DeriveKeys([]byte(vaultKey), salt, amnesia.KeyTypePassphrase)
	wrappedDEK, nonce, _ := amnesia.WrapKey(dek, kek)
	fullWrappedDEK := append(nonce, wrappedDEK...)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": base64.StdEncoding.EncodeToString(fullWrappedDEK),
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(
		WithAPIURL(server.URL),
		WithToken("token"),
		WithProjectID("prj_1"),
		WithVaultKey(vaultKey),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Zero should clear dek and hmacKey slices without panicking.
	c.Zero()

	for i, v := range c.dek {
		if v != 0 {
			t.Errorf("Expected dek[%d]=0 after Zero(), got %d", i, v)
		}
	}
	for i, v := range c.hmacKey {
		if v != 0 {
			t.Errorf("Expected hmacKey[%d]=0 after Zero(), got %d", i, v)
		}
	}
}

func TestClientAPI(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	salt := amnesia.GenerateSalt()
	kek, _ := amnesia.DeriveKeys([]byte(vaultKey), salt, amnesia.KeyTypePassphrase)
	wrappedDEK, nonce, _ := amnesia.WrapKey(dek, kek)
	fullWrappedDEK := append(nonce, wrappedDEK...)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": base64.StdEncoding.EncodeToString(fullWrappedDEK),
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(
		WithAPIURL(server.URL),
		WithToken("token"),
		WithProjectID("prj_1"),
		WithVaultKey(vaultKey),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// API() should return the underlying API client (non-nil).
	if c.API() == nil {
		t.Error("Expected API() to return non-nil client")
	}
}
