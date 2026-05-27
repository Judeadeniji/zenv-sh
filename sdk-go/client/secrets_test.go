package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ListSecrets(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"secrets": []map[string]string{
				{"id": "sec_1", "name_hash": "hash1"},
				{"id": "sec_2", "name_hash": "hash2"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	items, err := c.ListSecrets("prj_1", "env_1")
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
	if items[0].ID != "sec_1" {
		t.Errorf("Expected sec_1, got %s", items[0].ID)
	}
}

func TestClient_BulkFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		var req map[string][]string
		json.NewDecoder(r.Body).Decode(&req)
		
		if len(req["name_hashes"]) != 2 {
			t.Errorf("Expected 2 hashes, got %d", len(req["name_hashes"]))
		}

		resp := map[string]interface{}{
			"secrets": []map[string]string{
				{"id": "sec_1", "ciphertext": "ct1"},
				{"id": "sec_2", "ciphertext": "ct2"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	secrets, err := c.BulkFetch("prj_1", "env_1", []string{"hash1", "hash2"})
	if err != nil {
		t.Fatalf("BulkFetch failed: %v", err)
	}

	if len(secrets) != 2 {
		t.Errorf("Expected 2 secrets, got %d", len(secrets))
	}
}
