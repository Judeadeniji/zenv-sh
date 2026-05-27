package commands

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/sdk-go/client"
	"github.com/Judeadeniji/zenv-sh/cli/internal/config"
)

// gcmNonceSize is the AES-GCM nonce length prepended to all wrapped blobs.
const gcmNonceSize = 12

func newUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlock",
		Short: "Derive and save the project key using your Vault Key",
		Long: `Interactive one-time setup: unwrap the project vault key from your
key grant using your Vault Key (passphrase or PIN).

Use this on a trusted developer machine — not in CI. For pipelines, bots,
and other unattended environments, set ZENV_PROJECT_KEY from the dashboard
instead (store it as a CI secret).

Requires a saved service token (zenv login).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Token == "" {
				return fmt.Errorf("not authenticated.\nRun: zenv login")
			}
			return deriveProjectKey(api)
		},
	}
}

// deriveProjectKey fetches vault material and the key grant, prompts for the
// vault key, and derives + saves the project key.
func deriveProjectKey(apiClient *client.Client) error {
	info, err := apiClient.Whoami()
	if err != nil {
		return fmt.Errorf("whoami: %w", err)
	}
	projectID := info.ProjectID
	if projectID == "" {
		return fmt.Errorf("token has no project scope")
	}

	vault, err := apiClient.GetVaultMaterial()
	if err != nil {
		return fmt.Errorf("fetch vault material: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(vault.Salt)
	if err != nil {
		return fmt.Errorf("decode salt: %w", err)
	}

	vaultKey, err := PromptSecret("vault key")
	if err != nil {
		return fmt.Errorf("read vault key: %w", err)
	}
	defer zeroBytes(vaultKey)

	keyType := amnesia.KeyTypePassphrase
	if vault.VaultKeyType == "pin" {
		keyType = amnesia.KeyTypePIN
	}

	kek, authKey := amnesia.DeriveKeys(vaultKey, salt, keyType)
	defer zeroBytes(kek)
	defer zeroBytes(authKey)

	authKeyHash := base64.StdEncoding.EncodeToString(amnesia.HashAuthKey(authKey))
	if err := apiClient.VerifyVaultKey(authKeyHash); err != nil {
		return fmt.Errorf("Wrong Vault Key: %w", err)
	}

	wrappedDEK, err := base64.StdEncoding.DecodeString(vault.WrappedDEK)
	if err != nil {
		return fmt.Errorf("decode wrapped_dek: %w", err)
	}
	if len(wrappedDEK) <= gcmNonceSize {
		return fmt.Errorf("wrapped DEK too short")
	}

	dek, err := amnesia.UnwrapKey(wrappedDEK[gcmNonceSize:], wrappedDEK[:gcmNonceSize], kek)
	if err != nil {
		return fmt.Errorf("unwrap DEK (wrong vault key?): %w", err)
	}
	defer zeroBytes(dek)

	wrappedPrivateKey, err := base64.StdEncoding.DecodeString(vault.WrappedPrivateKey)
	if err != nil {
		return fmt.Errorf("decode wrapped_private_key: %w", err)
	}
	if len(wrappedPrivateKey) <= gcmNonceSize {
		return fmt.Errorf("wrapped private key too short")
	}

	privateKey, err := amnesia.Decrypt(wrappedPrivateKey[gcmNonceSize:], wrappedPrivateKey[:gcmNonceSize], dek)
	if err != nil {
		return fmt.Errorf("unwrap private key: %w", err)
	}
	defer zeroBytes(privateKey)

	grant, err := apiClient.GetKeyGrant(projectID)
	if err != nil {
		return fmt.Errorf("fetch key grant: %w", err)
	}

	wrappedProjectVaultKey, err := base64.StdEncoding.DecodeString(grant.WrappedProjectVaultKey)
	if err != nil {
		return fmt.Errorf("decode wrapped_project_vault_key: %w", err)
	}

	projectVaultKeyBytes, err := amnesia.UnwrapWithPrivateKey(wrappedProjectVaultKey, privateKey)
	if err != nil {
		return fmt.Errorf("unwrap project vault key: %w", err)
	}

	// config.SetForProject requires a string — this conversion is an acknowledged
	// tradeoff; the []byte is zeroed immediately after.
	projectKey := string(projectVaultKeyBytes)
	zeroBytes(projectVaultKeyBytes)

	if err := config.SetForProject(projectID, config.KeyProjectKey, projectKey); err != nil {
		return fmt.Errorf("save project_key: %w", err)
	}
	if err := config.SetLocal(config.KeyProject, projectID); err != nil {
		return fmt.Errorf("save project to .zenv: %w", err)
	}
	if info.Environment != "" {
		if err := config.SetLocal(config.KeyEnv, info.Environment); err != nil {
			return fmt.Errorf("save env to .zenv: %w", err)
		}
	}

	credPath, _ := config.ProjectCredentialsPath(projectID)
	fmt.Fprintf(os.Stderr, "Project key saved for this project:\n  %s\n\n", credPath)
	fmt.Fprintln(os.Stderr, "Each project has its own credentials. Switch repos with:")
	fmt.Fprintln(os.Stderr, "  zenv projects init <project-id>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "To store one key globally (all projects — not recommended):")
	fmt.Fprintln(os.Stderr, "  zenv config set --global project_key <key>")
	return nil
}
