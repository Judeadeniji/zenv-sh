package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CreateOrg(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["name"] != "acme" {
			t.Errorf("Expected name=acme, got %s", req["name"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(OrgResponse{ID: "org_1", Name: "acme", OwnerID: "user_1"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	org, err := c.CreateOrg("acme")
	if err != nil {
		t.Fatalf("CreateOrg failed: %v", err)
	}
	if org.ID != "org_1" {
		t.Errorf("Expected org_1, got %s", org.ID)
	}
	if org.Name != "acme" {
		t.Errorf("Expected acme, got %s", org.Name)
	}
}

func TestClient_CreateOrg_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "org already exists"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.CreateOrg("duplicate")
	if err == nil {
		t.Error("Expected error for conflict response")
	}
}

func TestClient_ListOrgs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		resp := map[string]interface{}{
			"organizations": []OrgResponse{
				{ID: "org_1", Name: "acme"},
				{ID: "org_2", Name: "globex"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	orgs, err := c.ListOrgs()
	if err != nil {
		t.Fatalf("ListOrgs failed: %v", err)
	}
	if len(orgs) != 2 {
		t.Errorf("Expected 2 orgs, got %d", len(orgs))
	}
	if orgs[0].ID != "org_1" {
		t.Errorf("Expected org_1, got %s", orgs[0].ID)
	}
}

func TestClient_ListOrgs_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.ListOrgs()
	if err == nil {
		t.Error("Expected error for unauthorized response")
	}
}

func TestClient_GetOrg(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs/org_1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(OrgResponse{ID: "org_1", Name: "acme", OwnerID: "user_1"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	org, err := c.GetOrg("org_1")
	if err != nil {
		t.Fatalf("GetOrg failed: %v", err)
	}
	if org.ID != "org_1" {
		t.Errorf("Expected org_1, got %s", org.ID)
	}
}

func TestClient_GetOrg_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs/org_missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "not found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.GetOrg("org_missing")
	if err == nil {
		t.Error("Expected error for not found response")
	}
}

func TestClient_ListMembers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs/org_1/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		resp := map[string]interface{}{
			"members": []MemberResponse{
				{ID: "mem_1", UserID: "user_1", Email: "alice@example.com", Role: "owner"},
				{ID: "mem_2", UserID: "user_2", Email: "bob@example.com", Role: "member"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	members, err := c.ListMembers("org_1")
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
	if members[0].Role != "owner" {
		t.Errorf("Expected role=owner, got %s", members[0].Role)
	}
}

func TestClient_AddMember(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs/org_1/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var req AddMemberRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.UserID != "user_3" || req.Role != "member" {
			t.Errorf("Unexpected request body: %+v", req)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(MemberResponse{ID: "mem_3", UserID: "user_3", Role: "member"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	m, err := c.AddMember("org_1", AddMemberRequest{UserID: "user_3", Role: "member"})
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if m.UserID != "user_3" {
		t.Errorf("Expected user_3, got %s", m.UserID)
	}
}

func TestClient_AddMember_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs/org_1/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "already a member"})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.AddMember("org_1", AddMemberRequest{UserID: "user_1", Role: "member"})
	if err == nil {
		t.Error("Expected error for conflict response")
	}
}

func TestClient_RemoveMember(t *testing.T) {
	removed := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs/org_1/members/mem_2", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		removed = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	err := c.RemoveMember("org_1", "mem_2")
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}
	if !removed {
		t.Error("Handler was not called")
	}
}

func TestClient_RemoveMember_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/orgs/org_1/members/mem_missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "member not found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	err := c.RemoveMember("org_1", "mem_missing")
	if err == nil {
		t.Error("Expected error for not found response")
	}
}