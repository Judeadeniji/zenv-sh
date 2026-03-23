package client

import (
	"encoding/json"
	"fmt"
)

// --- Organizations Types ---

type OrgResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OwnerID   string `json:"owner_id"`
	CreatedAt string `json:"created_at"`
}

type MemberResponse struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// --- Organizations ---

func (c *Client) CreateOrg(name string) (*OrgResponse, error) {
	body, status, err := c.post("/v1/sdk/orgs", map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	if status != 201 {
		return nil, parseError(body, status)
	}
	var o OrgResponse
	if err := json.Unmarshal(body, &o); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &o, nil
}

func (c *Client) ListOrgs() ([]OrgResponse, error) {
	body, status, err := c.get("/v1/sdk/orgs", nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var resp struct {
		Organizations []OrgResponse `json:"organizations"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Organizations, nil
}

func (c *Client) GetOrg(orgID string) (*OrgResponse, error) {
	body, status, err := c.get("/v1/sdk/orgs/"+orgID, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var o OrgResponse
	if err := json.Unmarshal(body, &o); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &o, nil
}

func (c *Client) ListMembers(orgID string) ([]MemberResponse, error) {
	body, status, err := c.get("/v1/sdk/orgs/"+orgID+"/members", nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, parseError(body, status)
	}
	var resp struct {
		Members []MemberResponse `json:"members"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return resp.Members, nil
}

func (c *Client) AddMember(orgID string, req AddMemberRequest) (*MemberResponse, error) {
	body, status, err := c.post("/v1/sdk/orgs/"+orgID+"/members", req)
	if err != nil {
		return nil, err
	}
	if status != 201 {
		return nil, parseError(body, status)
	}
	var m MemberResponse
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &m, nil
}

func (c *Client) RemoveMember(orgID, memberID string) error {
	body, status, err := c.delete("/v1/sdk/orgs/"+orgID+"/members/"+memberID, nil)
	if err != nil {
		return err
	}
	if status != 200 {
		return parseError(body, status)
	}
	return nil
}
