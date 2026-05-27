package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CreateToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var req TokenCreateRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.ProjectID != "prj_1" {
			t.Errorf("Expected project_id=prj_1, got %s", req.ProjectID)
		}
		if req.Name != "ci-token" {
			t.Errorf("Expected name=ci-token, got %s", req.Name)
		}
		if req.Environment != "production" {
			t.Errorf("Expected environment=production, got %s", req.Environment)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(TokenResponse{
			ID:          "tok_1",
			Token:       "zenv_secret_token_value",
			Name:        "ci-token",
			ProjectID:   "prj_1",
			Environment: "production",
			Permission:  "read",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	tok, err := c.CreateToken(TokenCreateRequest{
		ProjectID:   "prj_1",
		Name:        "ci-token",
		Environment: "production",
		Permission:  "read",
	})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if tok.ID != "tok_1" {
		t.Errorf("Expected tok_1, got %s", tok.ID)
	}
	if tok.Token != "zenv_secret_token_value" {
		t.Errorf("Expected token value returned on create, got %s", tok.Token)
	}
}

func TestClient_CreateToken_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid permission"})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.CreateToken(TokenCreateRequest{
		ProjectID:  "prj_1",
		Permission: "invalid",
	})
	if err == nil {
		t.Error("Expected error for invalid permission response")
	}
}

func TestClient_CreateToken_WithExpiry(t *testing.T) {
	expiry := "2026-12-31T00:00:00Z"
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/tokens", func(w http.ResponseWriter, r *http.Request) {
		var req TokenCreateRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.ExpiresAt == nil || *req.ExpiresAt != expiry {
			t.Errorf("Expected expires_at=%s, got %v", expiry, req.ExpiresAt)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(TokenResponse{
			ID:          "tok_2",
			Token:       "zenv_expiring_token",
			ProjectID:   "prj_1",
			Environment: "production",
			Permission:  "read",
			ExpiresAt:   &expiry,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	tok, err := c.CreateToken(TokenCreateRequest{
		ProjectID:   "prj_1",
		Name:        "expiring-token",
		Environment: "production",
		Permission:  "read",
		ExpiresAt:   &expiry,
	})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if tok.ExpiresAt == nil || *tok.ExpiresAt != expiry {
		t.Errorf("Expected expires_at=%s in response", expiry)
	}
}

func TestClient_ListTokens(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("project_id") != "prj_1" {
			t.Errorf("Expected project_id=prj_1, got %s", r.URL.Query().Get("project_id"))
		}
		resp := map[string]interface{}{
			"tokens": []TokenResponse{
				{ID: "tok_1", Name: "ci-token", ProjectID: "prj_1", Environment: "production"},
				{ID: "tok_2", Name: "deploy-token", ProjectID: "prj_1", Environment: "staging"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	tokens, err := c.ListTokens("prj_1")
	if err != nil {
		t.Fatalf("ListTokens failed: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Name != "ci-token" {
		t.Errorf("Expected ci-token, got %s", tokens[0].Name)
	}
	if tokens[1].Environment != "staging" {
		t.Errorf("Expected staging, got %s", tokens[1].Environment)
	}
}

func TestClient_ListTokens_Empty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"tokens": []TokenResponse{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	tokens, err := c.ListTokens("prj_empty")
	if err != nil {
		t.Fatalf("ListTokens failed: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("Expected 0 tokens, got %d", len(tokens))
	}
}

func TestClient_ListTokens_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/tokens", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "forbidden"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.ListTokens("prj_1")
	if err == nil {
		t.Error("Expected error for forbidden response")
	}
}

func TestClient_RevokeToken(t *testing.T) {
	revoked := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/tokens/tok_1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		revoked = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	err := c.RevokeToken("tok_1")
	if err != nil {
		t.Fatalf("RevokeToken failed: %v", err)
	}
	if !revoked {
		t.Error("Handler was not called")
	}
}

func TestClient_RevokeToken_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/tokens/tok_missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "token not found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	err := c.RevokeToken("tok_missing")
	if err == nil {
		t.Error("Expected error for not found response")
	}
}