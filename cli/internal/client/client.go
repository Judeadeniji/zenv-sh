package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is the HTTP client for the zEnv API.
type Client struct {
	baseURL    string
	token      string // service token (Bearer auth)
	httpClient *http.Client
}

// New creates a new API client.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- Request helpers ---

func (c *Client) get(path string, query url.Values) ([]byte, int, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, 0, err
	}
	return c.do(req)
}

func (c *Client) post(path string, body any) ([]byte, int, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) put(path string, body any) ([]byte, int, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest("PUT", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) delete(path string, query url.Values) ([]byte, int, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, 0, err
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return body, resp.StatusCode, nil
}

// --- API Types ---

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

type ErrorResponse struct {
	Error string `json:"error"`
}

// --- SDK Endpoints ---

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

// --- Project Crypto ---

type ProjectCrypto struct {
	ProjectSalt       string `json:"project_salt"`        // base64
	WrappedProjectDEK string `json:"wrapped_project_dek"` // base64
}

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

// --- Tokens ---

type TokenCreateRequest struct {
	ProjectID   string  `json:"project_id"`
	Name        string  `json:"name"`
	Environment string  `json:"environment"`
	Permission  string  `json:"permission"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

type TokenResponse struct {
	ID          string  `json:"id"`
	Token       string  `json:"token,omitempty"` // only on create
	Name        string  `json:"name"`
	ProjectID   string  `json:"project_id"`
	Environment string  `json:"environment"`
	Permission  string  `json:"permission"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	RevokedAt   *string `json:"revoked_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func (c *Client) CreateToken(req TokenCreateRequest) (*TokenResponse, error) {
	body, status, err := c.post("/v1/tokens", req)
	if err != nil {
		return nil, err
	}
	if status != 201 {
		return nil, parseError(body, status)
	}
	var t TokenResponse
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &t, nil
}

func (c *Client) ListTokens(projectID string) ([]TokenResponse, error) {
	q := url.Values{"project_id": {projectID}}
	body, status, err := c.get("/v1/tokens", q)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var resp struct {
		Tokens []TokenResponse `json:"tokens"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Tokens, nil
}

func (c *Client) RevokeToken(tokenID string) error {
	body, status, err := c.delete("/v1/tokens/"+tokenID, nil)
	if err != nil {
		return err
	}
	if status != 200 {
		return parseError(body, status)
	}
	return nil
}

// --- Projects ---

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

func (c *Client) CreateProject(req CreateProjectRequest) (*ProjectResponse, error) {
	body, status, err := c.post("/v1/projects", req)
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
	body, status, err := c.get("/v1/projects", q)
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
	body, status, err := c.get("/v1/projects/"+projectID, nil)
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

func parseError(body []byte, status int) error {
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("api error (%d): %s", status, errResp.Error)
	}
	return fmt.Errorf("api error (%d): %s", status, string(body))
}
