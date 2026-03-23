package commands

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/cli/internal/client"
	"github.com/Judeadeniji/zenv-sh/cli/internal/config"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with zEnv",
		Long: `Authenticate with zEnv by providing a service token.

Create a service token in the zEnv dashboard under Service Tokens,
then paste it here. The token is stored in ~/.config/zenv/credentials.

After saving the token, the CLI will prompt for your Vault Key to
automatically derive the Project Key from your key grant — no manual
copy-paste needed.

Run "zenv whoami" to verify your identity.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "Get a service token from your zEnv dashboard.")
			fmt.Fprintf(os.Stderr, "Dashboard: %s\n\n", cfg.AuthURL)

			reader := bufio.NewReader(os.Stdin)

			// Step 1: Collect service token
			fmt.Fprint(os.Stderr, "Paste your service token: ")
			token, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read token: %w", err)
			}
			token = strings.TrimSpace(token)

			if token == "" {
				return fmt.Errorf("no token provided")
			}
			if !strings.HasPrefix(token, "ze_") {
				return fmt.Errorf("invalid token format — service tokens start with ze_")
			}

			if err := config.Set(config.KeyToken, token); err != nil {
				return fmt.Errorf("save token: %w", err)
			}
			fmt.Fprintln(os.Stderr, "Token saved.")

			// Step 2: Try to auto-derive project key
			apiClient := client.New(cfg.APIURL, token)
			if err := deriveProjectKey(apiClient, reader); err != nil {
				// Non-fatal — fall back to manual setup
				fmt.Fprintf(os.Stderr, "\nCould not auto-derive project key: %s\n", err)
				fmt.Fprintln(os.Stderr, "You can set it manually:")
				fmt.Fprintln(os.Stderr, "  zenv config set --global project_key <your-project-key>")
			}

			fmt.Fprintln(os.Stderr, "\nRun `zenv whoami` to verify.")
			return nil
		},
	}
}

// deriveProjectKey fetches vault material and the key grant, prompts for the
// vault key, and derives + saves the project key automatically.
//
// Flow:
//  1. GET /sdk/whoami → project_id
//  2. GET /sdk/vault → salt, vault_key_type, wrapped_dek, wrapped_private_key
//  3. Prompt user for Vault Key
//  4. Argon2id(vault_key, salt) → KEK + auth_key
//  5. Unwrap DEK with KEK
//  6. Unwrap private key with DEK
//  7. GET /sdk/projects/{id}/key-grant → wrapped_project_vault_key
//  8. Unwrap project vault key with private key
//  9. Save project_key to credentials
func deriveProjectKey(apiClient *client.Client, reader *bufio.Reader) error {
	// Get token scope
	info, err := apiClient.Whoami()
	if err != nil {
		return fmt.Errorf("whoami: %w", err)
	}
	projectID := info.ProjectID
	if projectID == "" {
		return fmt.Errorf("token has no project scope")
	}

	// Fetch vault material
	vault, err := apiClient.GetVaultMaterial()
	if err != nil {
		return fmt.Errorf("fetch vault material: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(vault.Salt)
	if err != nil {
		return fmt.Errorf("decode salt: %w", err)
	}

	// Prompt for vault key
	fmt.Fprint(os.Stderr, "\nEnter your Vault Key to auto-configure the project key: ")
	vaultKeyInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read vault key: %w", err)
	}
	vaultKey := strings.TrimSpace(vaultKeyInput)
	if vaultKey == "" {
		return fmt.Errorf("no vault key provided")
	}

	// Map vault key type
	keyType := amnesia.KeyTypePassphrase
	if vault.VaultKeyType == "pin" {
		keyType = amnesia.KeyTypePIN
	}

	// Derive KEK from vault key + salt
	kek, _ := amnesia.DeriveKeys(vaultKey, salt, keyType)

	// Unwrap DEK
	wrappedDEK, err := base64.StdEncoding.DecodeString(vault.WrappedDEK)
	if err != nil {
		return fmt.Errorf("decode wrapped_dek: %w", err)
	}
	if len(wrappedDEK) < 13 {
		return fmt.Errorf("wrapped DEK too short")
	}
	dek, err := amnesia.UnwrapKey(wrappedDEK[12:], wrappedDEK[:12], kek)
	if err != nil {
		return fmt.Errorf("wrong Vault Key (unwrap DEK failed)")
	}

	// Unwrap private key
	wrappedPrivateKey, err := base64.StdEncoding.DecodeString(vault.WrappedPrivateKey)
	if err != nil {
		return fmt.Errorf("decode wrapped_private_key: %w", err)
	}
	if len(wrappedPrivateKey) < 13 {
		return fmt.Errorf("wrapped private key too short")
	}
	privateKey, err := amnesia.Decrypt(wrappedPrivateKey[12:], wrappedPrivateKey[:12], dek)
	if err != nil {
		return fmt.Errorf("unwrap private key: %w", err)
	}

	// Fetch key grant
	grant, err := apiClient.GetKeyGrant(projectID)
	if err != nil {
		return fmt.Errorf("fetch key grant: %w", err)
	}

	wrappedProjectVaultKey, err := base64.StdEncoding.DecodeString(grant.WrappedProjectVaultKey)
	if err != nil {
		return fmt.Errorf("decode wrapped_project_vault_key: %w", err)
	}

	// Unwrap project vault key with user's private key
	projectVaultKeyBytes, err := amnesia.UnwrapWithPrivateKey(wrappedProjectVaultKey, privateKey)
	if err != nil {
		return fmt.Errorf("unwrap project vault key: %w", err)
	}
	projectKey := string(projectVaultKeyBytes)

	// Save project key and project ID
	if err := config.Set(config.KeyProjectKey, projectKey); err != nil {
		return fmt.Errorf("save project_key: %w", err)
	}

	// Also set the project ID from the token if not already configured
	if config.Get(config.KeyProject) == "" {
		if err := config.Set(config.KeyProject, projectID); err != nil {
			return fmt.Errorf("save project: %w", err)
		}
	}

	// Set environment from token if not already configured
	if config.Get(config.KeyEnv) == "" && info.Environment != "" {
		if err := config.Set(config.KeyEnv, info.Environment); err != nil {
			return fmt.Errorf("save env: %w", err)
		}
	}

	fmt.Fprintln(os.Stderr, "Project key derived and saved.")
	return nil
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show your identity, token, and project scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				fmt.Fprintln(os.Stderr, "Not authenticated.")
				fmt.Fprintln(os.Stderr, "Run: zenv login")
				return nil
			}

			info, err := api.Whoami()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Authentication failed: %s\n", err)
				fmt.Fprintln(os.Stderr, "Run: zenv login")
				return nil
			}

			if info.UserName != "" {
				if info.UserEmail != "" {
					fmt.Fprintf(os.Stderr, "Logged in as %s <%s>\n\n", info.UserName, info.UserEmail)
				} else {
					fmt.Fprintf(os.Stderr, "Logged in as %s\n\n", info.UserName)
				}
			} else if info.UserEmail != "" {
				fmt.Fprintf(os.Stderr, "Logged in as %s\n\n", info.UserEmail)
			}

			fmt.Fprintf(os.Stderr, "Token:       %s\n", info.TokenName)
			if info.OrganizationName != "" {
				fmt.Fprintf(os.Stderr, "Org:         %s\n", info.OrganizationName)
			} else if info.OrganizationID != "" {
				fmt.Fprintf(os.Stderr, "Org:         %s\n", info.OrganizationID)
			}

			// Show the effective project from config, not the token's project.
			// The token may be scoped to one project but the CLI can operate on another.
			if cfg.Project != "" {
				// Try to resolve project name from the API
				if p, err := api.GetProject(cfg.Project); err == nil && p.Name != "" {
					fmt.Fprintf(os.Stderr, "Project:     %s\n", p.Name)
				} else {
					fmt.Fprintf(os.Stderr, "Project:     %s\n", cfg.Project)
				}
			} else if info.ProjectName != "" {
				fmt.Fprintf(os.Stderr, "Project:     %s\n", info.ProjectName)
			} else {
				fmt.Fprintf(os.Stderr, "Project:     %s\n", info.ProjectID)
			}

			// Show effective environment from config, fall back to token's.
			env := cfg.Env
			if env == "" {
				env = info.Environment
			}
			fmt.Fprintf(os.Stderr, "Environment: %s\n", env)
			fmt.Fprintf(os.Stderr, "Permission:  %s\n", info.Permission)

			if cfg.ProjectKey != "" {
				fmt.Fprintln(os.Stderr, "Vault Key:   set")
			} else {
				fmt.Fprintln(os.Stderr, "Vault Key:   (not set)")
			}

			return nil
		},
	}
}
