package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CreateSecret(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["project_id"] != "prj_1" {
			t.Errorf("Expected project_id=prj_1, got %s", req["project_id"])
		}
		if req["environment"] != "production" {
			t.Errorf("Expected environment=production, got %s", req["environment"])
		}
		if req["name_hash"] != "hash_abc" {
			t.Errorf("Expected name_hash=hash_abc, got %s", req["name_hash"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(SecretItem{
			ID:          "sec_1",
			ProjectID:   "prj_1",
			Environment: "production",
			NameHash:    "hash_abc",
			Ciphertext:  "enc_data",
			Nonce:       "nonce_val",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	item, err := c.CreateSecret("prj_1", "production", "hash_abc", "enc_data", "nonce_val")
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}
	if item.ID != "sec_1" {
		t.Errorf("Expected sec_1, got %s", item.ID)
	}
	if item.NameHash != "hash_abc" {
		t.Errorf("Expected hash_abc, got %s", item.NameHash)
	}
}

func TestClient_CreateSecret_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "secret already exists"})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.CreateSecret("prj_1", "production", "hash_abc", "enc_data", "nonce_val")
	if err == nil {
		t.Error("Expected error for conflict response")
	}
}

func TestClient_GetSecret(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/hash_abc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("project_id") != "prj_1" {
			t.Errorf("Expected project_id=prj_1, got %s", r.URL.Query().Get("project_id"))
		}
		if r.URL.Query().Get("environment") != "production" {
			t.Errorf("Expected environment=production, got %s", r.URL.Query().Get("environment"))
		}
		json.NewEncoder(w).Encode(SecretItem{
			ID:          "sec_1",
			ProjectID:   "prj_1",
			Environment: "production",
			NameHash:    "hash_abc",
			Ciphertext:  "enc_data",
			Nonce:       "nonce_val",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	item, err := c.GetSecret("prj_1", "production", "hash_abc")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if item.ID != "sec_1" {
		t.Errorf("Expected sec_1, got %s", item.ID)
	}
	if item.Ciphertext != "enc_data" {
		t.Errorf("Expected enc_data, got %s", item.Ciphertext)
	}
}

func TestClient_GetSecret_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/hash_missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "secret not found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.GetSecret("prj_1", "production", "hash_missing")
	if err == nil {
		t.Error("Expected error for not found response")
	}
}

func TestClient_BulkFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		if req["project_id"] != "prj_1" {
			t.Errorf("Expected project_id=prj_1, got %v", req["project_id"])
		}
		if req["environment"] != "production" {
			t.Errorf("Expected environment=production, got %v", req["environment"])
		}
		hashes, ok := req["name_hashes"].([]interface{})
		if !ok || len(hashes) != 2 {
			t.Errorf("Expected 2 name_hashes, got %v", req["name_hashes"])
		}
		resp := map[string]interface{}{
			"secrets": []SecretItem{
				{ID: "sec_1", NameHash: "hash_a", Ciphertext: "ct_a", Nonce: "nc_a"},
				{ID: "sec_2", NameHash: "hash_b", Ciphertext: "ct_b", Nonce: "nc_b"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	items, err := c.BulkFetch("prj_1", "production", []string{"hash_a", "hash_b"})
	if err != nil {
		t.Fatalf("BulkFetch failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
	if items[0].ID != "sec_1" {
		t.Errorf("Expected sec_1, got %s", items[0].ID)
	}
}

func TestClient_BulkFetch_Empty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"secrets": []SecretItem{},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	items, err := c.BulkFetch("prj_1", "production", []string{})
	if err != nil {
		t.Fatalf("BulkFetch failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
}

func TestClient_BulkFetch_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/bulk", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "forbidden"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.BulkFetch("prj_1", "production", []string{"hash_a"})
	if err == nil {
		t.Error("Expected error for forbidden response")
	}
}

func TestClient_ListSecrets(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("project_id") != "prj_1" {
			t.Errorf("Expected project_id=prj_1, got %s", r.URL.Query().Get("project_id"))
		}
		if r.URL.Query().Get("environment") != "staging" {
			t.Errorf("Expected environment=staging, got %s", r.URL.Query().Get("environment"))
		}
		resp := map[string]interface{}{
			"secrets": []ListItem{
				{ID: "sec_1", NameHash: "hash_a", Environment: "staging"},
				{ID: "sec_2", NameHash: "hash_b", Environment: "staging"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	items, err := c.ListSecrets("prj_1", "staging")
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
	if items[0].NameHash != "hash_a" {
		t.Errorf("Expected hash_a, got %s", items[0].NameHash)
	}
}

func TestClient_ListSecrets_Empty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"secrets": []ListItem{},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	items, err := c.ListSecrets("prj_1", "production")
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
}

func TestClient_ListSecrets_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.ListSecrets("prj_1", "production")
	if err == nil {
		t.Error("Expected error for unauthorized response")
	}
}

func TestClient_UpdateSecret(t *testing.T) {
	updated := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/hash_abc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		updated = true
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["ciphertext"] != "new_ct" {
			t.Errorf("Expected ciphertext=new_ct, got %s", req["ciphertext"])
		}
		json.NewEncoder(w).Encode(SecretItem{
			ID:         "sec_1",
			NameHash:   "hash_abc",
			Ciphertext: "new_ct",
			Nonce:      "new_nc",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	item, err := c.UpdateSecret("prj_1", "production", "hash_abc", "new_ct", "new_nc")
	if err != nil {
		t.Fatalf("UpdateSecret failed: %v", err)
	}
	if !updated {
		t.Error("Handler was not called")
	}
	if item.Ciphertext != "new_ct" {
		t.Errorf("Expected new_ct, got %s", item.Ciphertext)
	}
}

func TestClient_UpdateSecret_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/hash_abc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "secret not found"})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.UpdateSecret("prj_1", "production", "hash_abc", "new_ct", "new_nc")
	if err == nil {
		t.Error("Expected error for not found response")
	}
}

func TestClient_DeleteSecret(t *testing.T) {
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/hash_abc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Query().Get("project_id") != "prj_1" {
			t.Errorf("Expected project_id=prj_1, got %s", r.URL.Query().Get("project_id"))
		}
		deleted = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	err := c.DeleteSecret("prj_1", "production", "hash_abc")
	if err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}
	if !deleted {
		t.Error("Handler was not called")
	}
}

func TestClient_DeleteSecret_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/secrets/hash_missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "secret not found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	err := c.DeleteSecret("prj_1", "production", "hash_missing")
	if err == nil {
		t.Error("Expected error for not found response")
	}
}