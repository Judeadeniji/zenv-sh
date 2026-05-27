package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetProjectCrypto(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/crypto", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		resp := ProjectCrypto{
			ProjectSalt:       "c2FsdGJ5dGVz",
			WrappedProjectDEK: "d3JhcHBlZGRlaw==",
			VaultKeyType:      "passphrase",
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	pc, err := c.GetProjectCrypto("prj_1")
	if err != nil {
		t.Fatalf("GetProjectCrypto failed: %v", err)
	}
	if pc.ProjectSalt != "c2FsdGJ5dGVz" {
		t.Errorf("Expected project_salt c2FsdGJ5dGVz, got %s", pc.ProjectSalt)
	}
	if pc.VaultKeyType != "passphrase" {
		t.Errorf("Expected vault_key_type passphrase, got %s", pc.VaultKeyType)
	}
}

func TestClient_GetProjectCrypto_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_missing/crypto", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "project not found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.GetProjectCrypto("prj_missing")
	if err == nil {
		t.Error("Expected error for not found response")
	}
}

func TestClient_CreateProject(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var req CreateProjectRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name != "my-project" {
			t.Errorf("Expected name=my-project, got %s", req.Name)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ProjectResponse{
			ID:             "prj_new",
			OrganizationID: "org_1",
			Name:           "my-project",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	proj, err := c.CreateProject(CreateProjectRequest{
		OrganizationID:    "org_1",
		Name:              "my-project",
		ProjectSalt:       "salt_b64",
		WrappedProjectDEK: "wrapped_dek_b64",
	})
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	if proj.ID != "prj_new" {
		t.Errorf("Expected prj_new, got %s", proj.ID)
	}
}

func TestClient_CreateProject_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "forbidden"})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.CreateProject(CreateProjectRequest{Name: "x"})
	if err == nil {
		t.Error("Expected error for forbidden response")
	}
}

func TestClient_ListProjects(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("organization_id") != "org_1" {
			t.Errorf("Expected organization_id=org_1, got %s", r.URL.Query().Get("organization_id"))
		}
		resp := map[string]interface{}{
			"projects": []ProjectResponse{
				{ID: "prj_1", Name: "alpha"},
				{ID: "prj_2", Name: "beta"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	projects, err := c.ListProjects("org_1")
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
	if projects[1].Name != "beta" {
		t.Errorf("Expected beta, got %s", projects[1].Name)
	}
}

func TestClient_GetProject(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(ProjectResponse{ID: "prj_1", Name: "alpha", OrganizationID: "org_1"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	proj, err := c.GetProject("prj_1")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if proj.ID != "prj_1" {
		t.Errorf("Expected prj_1, got %s", proj.ID)
	}
}

func TestClient_GetProject_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "project not found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.GetProject("prj_missing")
	if err == nil {
		t.Error("Expected error for not found")
	}
}

func TestClient_GetVaultMaterial(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/vault", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(VaultMaterialResponse{
			Salt:         "saltval",
			VaultKeyType: "passphrase",
			WrappedDEK:   "wrappeddek",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	vm, err := c.GetVaultMaterial()
	if err != nil {
		t.Fatalf("GetVaultMaterial failed: %v", err)
	}
	if vm.VaultKeyType != "passphrase" {
		t.Errorf("Expected passphrase, got %s", vm.VaultKeyType)
	}
}

func TestClient_GetVaultMaterial_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/vault", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unauthorized"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.GetVaultMaterial()
	if err == nil {
		t.Error("Expected error for unauthorized")
	}
}

func TestClient_GetKeyGrant(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/key-grant", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(KeyGrantResponse{WrappedProjectVaultKey: "encryptedvaultkey"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	kg, err := c.GetKeyGrant("prj_1")
	if err != nil {
		t.Fatalf("GetKeyGrant failed: %v", err)
	}
	if kg.WrappedProjectVaultKey != "encryptedvaultkey" {
		t.Errorf("Expected encryptedvaultkey, got %s", kg.WrappedProjectVaultKey)
	}
}

func TestClient_GetKeyGrant_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sdk/projects/prj_1/key-grant", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "key grant not allowed"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(server.URL, "token")
	_, err := c.GetKeyGrant("prj_1")
	if err == nil {
		t.Error("Expected error for forbidden")
	}
}