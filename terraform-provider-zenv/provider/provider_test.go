package provider

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/sdk-go/crypto"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"zenv": providerserver.NewProtocol6WithError(New()()),
}

// setupMockAPI starts a mock zEnv server and returns its URL, a cleanup func,
// and the required vault key. It injects specific test secrets into the mock.
func setupMockAPI(t *testing.T) (url string, vaultKey string, teardown func()) {
	vaultKey = "test-vault-key-very-secure"
	dek := amnesia.GenerateKey()
	salt := amnesia.GenerateSalt()
	kek, _ := amnesia.DeriveKeys([]byte(vaultKey), salt, amnesia.KeyTypePassphrase)
	wrappedDEK, nonce, _ := amnesia.WrapKey(dek, kek)
	fullWrappedDEK := append(nonce, wrappedDEK...)

	// Setup our mock state
	// Secret 1: API_KEY -> secret-api-key
	// Secret 2: DB_PASS -> secret-db-pass
	hash1 := crypto.ComputeNameHash("API_KEY", dek)
	ct1, n1, h1, _ := crypto.EncryptSecret("API_KEY", "secret-api-key", dek, dek)

	hash2 := crypto.ComputeNameHash("DB_PASS", dek)
	ct2, n2, h2, _ := crypto.EncryptSecret("DB_PASS", "secret-db-pass", dek, dek)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_test/crypto", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"project_salt":        base64.StdEncoding.EncodeToString(salt),
			"wrapped_project_dek": base64.StdEncoding.EncodeToString(fullWrappedDEK),
		})
	})

	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"secrets": []map[string]string{
				{"id": "sec_1", "name_hash": hash1},
				{"id": "sec_2", "name_hash": hash2},
			},
		})
	})

	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		var req map[string][]string
		json.NewDecoder(r.Body).Decode(&req)

		secrets := []map[string]string{}
		for _, h := range req["name_hashes"] {
			if h == hash1 {
				secrets = append(secrets, map[string]string{"id": "sec_1", "ciphertext": ct1, "nonce": n1, "name_hash": h1})
			} else if h == hash2 {
				secrets = append(secrets, map[string]string{"id": "sec_2", "ciphertext": ct2, "nonce": n2, "name_hash": h2})
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"secrets": secrets,
		})
	})

	server := httptest.NewServer(mux)
	return server.URL, vaultKey, server.Close
}
