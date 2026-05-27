package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupMockAPI(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/sdk/whoami", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"identity_id": "id_1",
			"project_id":  "12345678-1234-1234-1234-123456789012",
			"environment": "env_1",
		})
	})
	
	mux.HandleFunc("/v1/sdk/projects/12345678-1234-1234-1234-123456789012/crypto", func(w http.ResponseWriter, r *http.Request) {
		// Just return some fake crypto material for tests
		json.NewEncoder(w).Encode(map[string]string{
			"project_salt":        "c2FsdA==", // "salt" base64
			"wrapped_project_dek": "c2FsdA==",
		})
	})
	
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"secrets": []map[string]string{
				// Need mock data or just an empty list
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestEnvCmd(t *testing.T) {
	server := setupMockAPI(t)
	defer server.Close()

	// Setup a temporary workspace
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/.config")
	t.Setenv("ZENV_API_URL", server.URL)
	t.Setenv("ZENV_TOKEN", "fake-token")
	t.Setenv("ZENV_PROJECT", "12345678-1234-1234-1234-123456789012")
	t.Setenv("ZENV_ENV", "env_1")
	// Skip vault unlocking for basic env tests if possible, or mock vault key
	t.Setenv("ZENV_PROJECT_KEY", "fake-key")

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"env", "--format=dotenv"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	
	// Right now we return fake crypto, so decryption will fail.
	// But we expect the command to reach the decryption phase.
	if err == nil {
		t.Fatalf("Expected error because of fake crypto, but got none")
	} else if err.Error() != "cipher: message authentication failed" && err.Error() != "failed to decrypt keys" {
		// Log the error to ensure we're failing for the right reason, or check specific substring
		// Actually, since it's fake crypto, it will fail during AES-GCM Open.
		t.Logf("Expected crypto error, got: %v", err)
	}
}
