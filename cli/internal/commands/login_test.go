package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupLoginMockAPI(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/whoami", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ze_valid_token") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"identity_id": "id_1",
			"project_id":  "12345678-1234-1234-1234-123456789012",
			"environment": "env_1",
		})
	})
	return httptest.NewServer(mux)
}

func TestLoginCmd(t *testing.T) {
	server := setupLoginMockAPI(t)
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("ZENV_CONFIG_DIR", tmpDir+"/.config/zenv")
	t.Setenv("ZENV_API_URL", server.URL)

	// Hijack Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.Write([]byte("ze_valid_token\n"))
		w.Close()
	}()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"login"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Verify that token is saved properly in HOME dir config
	// Because config.SetForProject should write to ~/.config/zenv/projects/12345678-1234-1234-1234-123456789012/credentials
	credPath := tmpDir + "/.config/zenv/projects/12345678-1234-1234-1234-123456789012/credentials"
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		t.Errorf("Credentials file was not created at %s", credPath)
	}
}
