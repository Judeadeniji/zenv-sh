package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// --- Secrets Types ---

type SecretItem struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
	NameHash    string `json:"name_hash"`
	Ciphertext  string `json:"ciphertext"`
	Nonce       string `json:"nonce"`
	Version     int    `json:"version"`
	UpdatedAt   string `json:"updated_at"`
}

type ListItem struct {
	ID          string `json:"id"`
	NameHash    string `json:"name_hash"`
	Environment string `json:"environment"`
	Version     int    `json:"version"`
	UpdatedAt   string `json:"updated_at"`
}

// --- Secrets ---

// CreateSecret stores an encrypted secret via the SDK endpoint.
func (c *Client) CreateSecret(projectID, env, nameHash, ciphertext, nonce string) (*SecretItem, error) {
	body, status, err := c.post("/v1/sdk/secrets", map[string]string{
		"project_id":  projectID,
		"environment": env,
		"name_hash":   nameHash,
		"ciphertext":  ciphertext,
		"nonce":       nonce,
	})
	if err != nil {
		return nil, err
	}
	if status != 201 {
		return nil, parseError(body, status)
	}

	var item SecretItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &item, nil
}

// GetSecret retrieves a single encrypted secret.
func (c *Client) GetSecret(projectID, env, nameHash string) (*SecretItem, error) {
	q := url.Values{"project_id": {projectID}, "environment": {env}}
	body, status, err := c.get("/v1/sdk/secrets/"+nameHash, q)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}

	var item SecretItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &item, nil
}

// BulkFetch retrieves multiple secrets by name hashes.
func (c *Client) BulkFetch(projectID, env string, nameHashes []string) ([]SecretItem, error) {
	body, status, err := c.post("/v1/sdk/secrets/bulk", map[string]any{
		"project_id":  projectID,
		"environment": env,
		"name_hashes": nameHashes,
	})
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}

	var resp struct {
		Secrets []SecretItem `json:"secrets"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Secrets, nil
}

// ListSecrets returns metadata for all secrets in a project+environment.
func (c *Client) ListSecrets(projectID, env string) ([]ListItem, error) {
	q := url.Values{"project_id": {projectID}, "environment": {env}}
	body, status, err := c.get("/v1/sdk/secrets", q)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}

	var resp struct {
		Secrets []ListItem `json:"secrets"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Secrets, nil
}

// UpdateSecret updates an existing secret.
func (c *Client) UpdateSecret(projectID, env, nameHash, ciphertext, nonce string) (*SecretItem, error) {
	q := url.Values{"project_id": {projectID}, "environment": {env}}
	body, status, err := c.put("/v1/sdk/secrets/"+nameHash+"?"+q.Encode(), map[string]string{
		"ciphertext": ciphertext,
		"nonce":      nonce,
	})
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}

	var item SecretItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &item, nil
}

// DeleteSecret deletes a secret.
func (c *Client) DeleteSecret(projectID, env, nameHash string) error {
	q := url.Values{"project_id": {projectID}, "environment": {env}}
	body, status, err := c.delete("/v1/sdk/secrets/"+nameHash, q)
	if err != nil {
		return err
	}
	if status != 200 {
		return parseError(body, status)
	}
	return nil
}
