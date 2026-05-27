package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// --- Tokens Types ---

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

// --- Tokens ---

func (c *Client) CreateToken(req TokenCreateRequest) (*TokenResponse, error) {
	body, status, err := c.post("/v1/sdk/tokens", req)
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
	body, status, err := c.get("/v1/sdk/tokens", q)
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
	body, status, err := c.delete("/v1/sdk/tokens/"+tokenID, nil)
	if err != nil {
		return err
	}
	if status != 200 {
		return parseError(body, status)
	}
	return nil
}
