package zenv

import (
	"encoding/base64"
	"fmt"
	"os"

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

type clientOptions struct {
	apiURL    string
	token     string
	projectID string
	vaultKey  string
}

// ClientOption configures the zEnv client.
type ClientOption func(*clientOptions)

// WithAPIURL configures a custom zEnv API URL (default: https://api.zenv.dev).
func WithAPIURL(url string) ClientOption {
	return func(o *clientOptions) { o.apiURL = url }
}

// WithToken configures the service token for authentication.
func WithToken(token string) ClientOption {
	return func(o *clientOptions) { o.token = token }
}

// WithProjectID configures the target project UUID.
func WithProjectID(projectID string) ClientOption {
	return func(o *clientOptions) { o.projectID = projectID }
}

// WithVaultKey configures the zero-knowledge vault key used to decrypt the project DEK.
func WithVaultKey(vaultKey string) ClientOption {
	return func(o *clientOptions) { o.vaultKey = vaultKey }
}

// NewClient initializes a zEnv SDK client, authenticates against the zEnv API,
// and derives the cryptographic keys required for zero-knowledge decryption.
// If options are not provided, it falls back to the ZENV_API_URL, ZENV_TOKEN, 
// ZENV_PROJECT, and ZENV_PROJECT_KEY environment variables.
func NewClient(opts ...ClientOption) (*Client, error) {
	o := &clientOptions{
		apiURL:    os.Getenv("ZENV_API_URL"),
		token:     os.Getenv("ZENV_TOKEN"),
		projectID: os.Getenv("ZENV_PROJECT"),
		vaultKey:  os.Getenv("ZENV_PROJECT_KEY"),
	}
	
	for _, opt := range opts {
		opt(o)
	}

	if o.apiURL == "" {
		o.apiURL = "https://api.zenv.dev"
	}
	if o.token == "" {
		return nil, fmt.Errorf("zenv: missing token (provide via WithToken or ZENV_TOKEN)")
	}
	if o.projectID == "" {
		return nil, fmt.Errorf("zenv: missing project ID (provide via WithProjectID or ZENV_PROJECT)")
	}
	if o.vaultKey == "" {
		return nil, fmt.Errorf("zenv: missing vault key (provide via WithVaultKey or ZENV_PROJECT_KEY)")
	}

	apiClient := client.New(o.apiURL, o.token)

	// Fetch project crypto
	pc, err := apiClient.GetProjectCrypto(o.projectID)
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

	projectKEK, _ := amnesia.DeriveKeys([]byte(o.vaultKey), projectSalt, amnesia.KeyTypePassphrase)

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
		projectID: o.projectID,
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
