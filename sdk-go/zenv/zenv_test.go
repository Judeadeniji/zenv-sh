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

	client, err := NewClient(server.URL, "token", "prj_1", vaultKey)
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
