package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Whoami(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/whoami", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer service-token" {
			t.Errorf("Missing or incorrect Authorization header: %s", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(WhoamiResponse{
			TokenName:        "ci-token",
			ProjectName:      "my-project",
			ProjectID:        "prj_1",
			OrganizationID:   "org_1",
			OrganizationName: "acme",
			Environment:      "production",
			Permission:       "read",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "service-token")
	resp, err := c.Whoami()
	if err != nil {
		t.Fatalf("Whoami failed: %v", err)
	}
	if resp.ProjectID != "prj_1" {
		t.Errorf("Expected ProjectID=prj_1, got %s", resp.ProjectID)
	}
	if resp.Environment != "production" {
		t.Errorf("Expected Environment=production, got %s", resp.Environment)
	}
	if resp.Permission != "read" {
		t.Errorf("Expected Permission=read, got %s", resp.Permission)
	}
	if resp.TokenName != "ci-token" {
		t.Errorf("Expected TokenName=ci-token, got %s", resp.TokenName)
	}
	if resp.OrganizationName != "acme" {
		t.Errorf("Expected OrganizationName=acme, got %s", resp.OrganizationName)
	}
}

func TestClient_Whoami_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/whoami", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid token"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "bad-token")
	_, err := c.Whoami()
	if err == nil {
		t.Error("Expected error for unauthorized response")
	}
}

func TestClient_Whoami_UserEmailOptional(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/whoami", func(w http.ResponseWriter, r *http.Request) {
		// Omit optional fields user_name and user_email (service token scenario)
		json.NewEncoder(w).Encode(WhoamiResponse{
			TokenName:   "svc-token",
			ProjectName: "proj",
			ProjectID:   "prj_2",
			Environment: "staging",
			Permission:  "write",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	resp, err := c.Whoami()
	if err != nil {
		t.Fatalf("Whoami failed: %v", err)
	}
	if resp.UserEmail != "" {
		t.Errorf("Expected empty UserEmail for service token, got %s", resp.UserEmail)
	}
	if resp.UserName != "" {
		t.Errorf("Expected empty UserName for service token, got %s", resp.UserName)
	}
	if resp.ProjectID != "prj_2" {
		t.Errorf("Expected prj_2, got %s", resp.ProjectID)
	}
}

func TestClient_Whoami_UserFields(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/whoami", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(WhoamiResponse{
			UserName:    "alice",
			UserEmail:   "alice@example.com",
			TokenName:   "personal",
			ProjectName: "my-project",
			ProjectID:   "prj_3",
			Environment: "development",
			Permission:  "write",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "user-token")
	resp, err := c.Whoami()
	if err != nil {
		t.Fatalf("Whoami failed: %v", err)
	}
	if resp.UserName != "alice" {
		t.Errorf("Expected UserName=alice, got %s", resp.UserName)
	}
	if resp.UserEmail != "alice@example.com" {
		t.Errorf("Expected UserEmail=alice@example.com, got %s", resp.UserEmail)
	}
}