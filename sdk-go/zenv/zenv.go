package zenv

import (
	"encoding/base64"
	"fmt"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/sdk-go/client"
	"github.com/Judeadeniji/zenv-sh/sdk-go/crypto"
)

// Client is the high-level zEnv SDK client.
type Client struct {
	apiClient *client.Client
	projectID string
	dek       []byte
	hmacKey   []byte
}

// NewClient initializes a zEnv SDK client, authenticates against the zEnv API,
// and derives the cryptographic keys required for zero-knowledge decryption.
func NewClient(apiURL, token, projectID, vaultKey string) (*Client, error) {
	if apiURL == "" {
		apiURL = "https://api.zenv.dev"
	}
	apiClient := client.New(apiURL, token)

	// Fetch project crypto
	pc, err := apiClient.GetProjectCrypto(projectID)
	if err != nil {
		return nil, fmt.Errorf("fetch project crypto: %w", err)
	}

	projectSalt, err := base64.StdEncoding.DecodeString(pc.ProjectSalt)
	if err != nil {
		return nil, fmt.Errorf("decode project_salt: %w", err)
	}
	wrappedProjectDEK, err := base64.StdEncoding.DecodeString(pc.WrappedProjectDEK)
	if err != nil {
		return nil, fmt.Errorf("decode wrapped_project_dek: %w", err)
	}

	projectKEK, _ := amnesia.DeriveKeys([]byte(vaultKey), projectSalt, amnesia.KeyTypePassphrase)

	if len(wrappedProjectDEK) < 13 {
		return nil, fmt.Errorf("wrapped project DEK too short")
	}
	nonce := wrappedProjectDEK[:12]
	ciphertext := wrappedProjectDEK[12:]

	projectDEK, err := amnesia.UnwrapKey(ciphertext, nonce, projectKEK)
	if err != nil {
		return nil, fmt.Errorf("unwrap project DEK: %w", err)
	}

	return &Client{
		apiClient: apiClient,
		projectID: projectID,
		dek:       projectDEK,
		hmacKey:   projectDEK,
	}, nil
}

// API returns the underlying REST API client for direct interactions.
func (c *Client) API() *client.Client {
	return c.apiClient
}

// FetchSecret securely retrieves and decrypts a specific secret by name.
func (c *Client) FetchSecret(env, name string) (string, error) {
	hash := crypto.ComputeNameHash(name, c.hmacKey)
	
	items, err := c.apiClient.BulkFetch(c.projectID, env, []string{hash})
	if err != nil {
		return "", fmt.Errorf("fetch secret: %w", err)
	}
	if len(items) == 0 {
		return "", fmt.Errorf("secret not found")
	}
	
	payload, err := crypto.DecryptSecret(items[0].Ciphertext, items[0].Nonce, c.dek)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return payload.Value, nil
}

// FetchAllSecrets securely retrieves and decrypts all secrets in an environment.
func (c *Client) FetchAllSecrets(env string) (map[string]string, error) {
	items, err := c.apiClient.ListSecrets(c.projectID, env)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	
	if len(items) == 0 {
		return map[string]string{}, nil
	}
	
	hashes := make([]string, len(items))
	for i, item := range items {
		hashes[i] = item.NameHash
	}
	
	secrets, err := c.apiClient.BulkFetch(c.projectID, env, hashes)
	if err != nil {
		return nil, fmt.Errorf("bulk fetch: %w", err)
	}
	
	result := make(map[string]string, len(secrets))
	for _, s := range secrets {
		payload, err := crypto.DecryptSecret(s.Ciphertext, s.Nonce, c.dek)
		if err != nil {
			return nil, fmt.Errorf("decrypt secret %s: %w", s.ID, err)
		}
		result[payload.Name] = payload.Value
	}
	
	return result, nil
}

// Zero explicitly zeros out the derived cryptographic keys from memory.
func (c *Client) Zero() {
	for i := range c.dek {
		c.dek[i] = 0
	}
	for i := range c.hmacKey {
		c.hmacKey[i] = 0
	}
}
