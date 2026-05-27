package client

import (
	"encoding/json"
	"fmt"
)

type WhoamiResponse struct {
	UserName         string `json:"user_name,omitempty"`
	UserEmail        string `json:"user_email,omitempty"`
	TokenName        string `json:"token_name"`
	ProjectName      string `json:"project_name"`
	ProjectID        string `json:"project_id"`
	OrganizationID   string `json:"organization_id,omitempty"`
	OrganizationName string `json:"organization_name,omitempty"`
	Environment      string `json:"environment"`
	Permission       string `json:"permission"`
}

func (c *Client) Whoami() (*WhoamiResponse, error) {
	body, status, err := c.get("/v1/sdk/whoami", nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var resp WhoamiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}
