package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// --- Projects Types ---

type ProjectCrypto struct {
	ProjectSalt       string `json:"project_salt"`              // base64
	WrappedProjectDEK string `json:"wrapped_project_dek"`       // base64
	VaultKeyType      string `json:"vault_key_type,omitempty"`  // "pin" or "passphrase"
}

type ProjectResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	CreatedAt      string `json:"created_at"`
}

type CreateProjectRequest struct {
	OrganizationID         string `json:"organization_id"`
	Name                   string `json:"name"`
	ProjectSalt            string `json:"project_salt"`
	WrappedProjectDEK      string `json:"wrapped_project_dek"`
	WrappedProjectVaultKey string `json:"wrapped_project_vault_key"`
}

// --- Projects ---

// GetProjectCrypto fetches the project salt and wrapped DEK for key derivation.
func (c *Client) GetProjectCrypto(projectID string) (*ProjectCrypto, error) {
	body, status, err := c.get("/v1/sdk/projects/"+projectID+"/crypto", nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}

	var pc ProjectCrypto
	if err := json.Unmarshal(body, &pc); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &pc, nil
}

func (c *Client) CreateProject(req CreateProjectRequest) (*ProjectResponse, error) {
	body, status, err := c.post("/v1/sdk/projects", req)
	if err != nil {
		return nil, err
	}
	if status != 201 {
		return nil, parseError(body, status)
	}
	var p ProjectResponse
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &p, nil
}

func (c *Client) ListProjects(orgID string) ([]ProjectResponse, error) {
	q := url.Values{"organization_id": {orgID}}
	body, status, err := c.get("/v1/sdk/projects", q)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var resp struct {
		Projects []ProjectResponse `json:"projects"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Projects, nil
}

func (c *Client) GetProject(projectID string) (*ProjectResponse, error) {
	body, status, err := c.get("/v1/sdk/projects/"+projectID, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var p ProjectResponse
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &p, nil
}

// --- Vault Material ---

type VaultMaterialResponse struct {
	Salt              string `json:"salt"`
	VaultKeyType      string `json:"vault_key_type"`
	WrappedDEK        string `json:"wrapped_dek"`
	WrappedPrivateKey string `json:"wrapped_private_key"`
	PublicKey         string `json:"public_key"`
}

func (c *Client) GetVaultMaterial() (*VaultMaterialResponse, error) {
	body, status, err := c.get("/v1/sdk/vault", nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var resp VaultMaterialResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}

// --- Key Grant ---

type KeyGrantResponse struct {
	WrappedProjectVaultKey string `json:"wrapped_project_vault_key"`
}

func (c *Client) GetKeyGrant(projectID string) (*KeyGrantResponse, error) {
	body, status, err := c.get("/v1/sdk/projects/"+projectID+"/key-grant", nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var resp KeyGrantResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}
