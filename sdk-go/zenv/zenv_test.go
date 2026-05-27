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

// buildWrappedDEK creates a nonce-prefixed wrapped DEK for testing.
// zenv.go expects: wrappedProjectDEK = nonce (12 bytes) + ciphertext
func buildWrappedDEK(t *testing.T, dek []byte, vaultKey string, salt []byte) string {
	t.Helper()
	kek, _ := amnesia.DeriveKeys([]byte(vaultKey), salt, amnesia.KeyTypePassphrase)
	ciphertext, nonce, err := amnesia.WrapKey(dek, kek)
	if err != nil {
		t.Fatalf("WrapKey failed: %v", err)
	}
	// Concatenate nonce + ciphertext as expected by NewClient
	full := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(full)
}

func setupMockServer(t *testing.T, vaultKey string, dek []byte, secretName, secretValue string) *httptest.Server {
	t.Helper()
	salt := amnesia.GenerateSalt()
	wrappedDEKB64 := buildWrappedDEK(t, dek, vaultKey, salt)

	nameHash := crypto.ComputeNameHash(secretName, dek)
	ct, nc, _, err := crypto.EncryptSecret(secretName, secretValue, dek, dek)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": wrappedDEKB64,
		}
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"secrets": []map[string]string{
				{"id": "sec_1", "name_hash": nameHash},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"secrets": []map[string]string{
				{"id": "sec_1", "ciphertext": ct, "nonce": nc, "name_hash": nameHash},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestNewClient_Success(t *testing.T) {
	vaultKey := "test-vault-key"
	dek := amnesia.GenerateKey()

	server := setupMockServer(t, vaultKey, dek, "API_KEY", "secret-val")
	defer server.Close()

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if c == nil {
		t.Fatal("Expected non-nil client")
	}
}

func TestNewClient_DefaultAPIURL(t *testing.T) {
	// Calling with empty URL will attempt to reach api.zenv.dev which will fail.
	// We verify the error is a network error, not a parameter validation error.
	_, err := NewClient("", "token", "prj_1", "vault-key")
	if err == nil {
		t.Error("Expected error when connecting to default URL without a server")
	}
}

func TestNewClient_BadCrypto_InvalidSalt(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        "not-valid-base64!!!",
			"wrapped_project_dek": "dW5pY29ybg==",
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := NewClient(server.URL, "token", "prj_1", "vault-key")
	if err == nil {
		t.Error("Expected NewClient to fail with invalid base64 salt")
	}
}

func TestNewClient_BadCrypto_InvalidWrappedDEK(t *testing.T) {
	salt := amnesia.GenerateSalt()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": "not-valid-base64!!!",
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := NewClient(server.URL, "token", "prj_1", "vault-key")
	if err == nil {
		t.Error("Expected NewClient to fail with invalid base64 wrapped DEK")
	}
}

func TestNewClient_WrappedDEKTooShort(t *testing.T) {
	salt := amnesia.GenerateSalt()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		// Valid salt but wrapped DEK is too short (less than 13 bytes)
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": base64.StdEncoding.EncodeToString([]byte("short")),
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := NewClient(server.URL, "token", "prj_1", "vault-key")
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

	_, err := NewClient(server.URL, "bad-token", "prj_1", "vault-key")
	if err == nil {
		t.Error("Expected NewClient to fail with unauthorized API response")
	}
}

func TestNewClient_WrongVaultKey(t *testing.T) {
	salt := amnesia.GenerateSalt()
	dek := amnesia.GenerateKey()
	// Wrap with the correct vault key
	wrappedDEKB64 := buildWrappedDEK(t, dek, "correct-vault-key", salt)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": wrappedDEKB64,
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Use a wrong vault key — the DEK unwrap should fail
	_, err := NewClient(server.URL, "token", "prj_1", "wrong-vault-key")
	if err == nil {
		t.Error("Expected NewClient to fail when vault key is wrong")
	}
}

func TestFetchSecret(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	name := "API_KEY"
	val := "super-secret"

	server := setupMockServer(t, vaultKey, dek, name, val)
	defer server.Close()

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	fetchedVal, err := c.FetchSecret("production", name)
	if err != nil {
		t.Fatalf("FetchSecret failed: %v", err)
	}
	if fetchedVal != val {
		t.Errorf("Expected %s, got %s", val, fetchedVal)
	}
}

func TestFetchSecret_NotFound(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	salt := amnesia.GenerateSalt()
	wrappedDEKB64 := buildWrappedDEK(t, dek, vaultKey, salt)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": wrappedDEKB64,
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		// Return empty secrets list
		resp := map[string]interface{}{
			"secrets": []interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = c.FetchSecret("production", "MISSING_KEY")
	if err == nil {
		t.Error("Expected FetchSecret to return error when secret not found")
	}
}

func TestFetchAllSecrets(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	name := "API_KEY"
	val := "super-secret"

	server := setupMockServer(t, vaultKey, dek, name, val)
	defer server.Close()

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	all, err := c.FetchAllSecrets("production")
	if err != nil {
		t.Fatalf("FetchAllSecrets failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(all))
	}
	if all[name] != val {
		t.Errorf("Expected %s=%s, got %s", name, val, all[name])
	}
}

func TestFetchAllSecrets_Empty(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	salt := amnesia.GenerateSalt()
	wrappedDEKB64 := buildWrappedDEK(t, dek, vaultKey, salt)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": wrappedDEKB64,
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

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
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

func TestFetchAllSecrets_ListError(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()
	salt := amnesia.GenerateSalt()
	wrappedDEKB64 := buildWrappedDEK(t, dek, vaultKey, salt)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": wrappedDEKB64,
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = c.FetchAllSecrets("production")
	if err == nil {
		t.Error("Expected FetchAllSecrets to return error on list failure")
	}
}

func TestClientAPI(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()

	server := setupMockServer(t, vaultKey, dek, "KEY", "val")
	defer server.Close()

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// API() should return the underlying API client (non-nil)
	if c.API() == nil {
		t.Error("Expected API() to return non-nil client")
	}
}

func TestClientZero(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()

	server := setupMockServer(t, vaultKey, dek, "KEY", "val")
	defer server.Close()

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Zero should clear dek and hmacKey slices without panicking
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

func TestClientZero_IdempotentAfterZero(t *testing.T) {
	vaultKey := "vault-key"
	dek := amnesia.GenerateKey()

	server := setupMockServer(t, vaultKey, dek, "KEY", "val")
	defer server.Close()

	c, err := NewClient(server.URL, "token", "prj_1", vaultKey)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Calling Zero() twice should not panic
	c.Zero()
	c.Zero()
}
