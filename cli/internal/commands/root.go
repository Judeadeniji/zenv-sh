package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/cli/internal/client"
	"github.com/Judeadeniji/zenv-sh/cli/internal/config"
)

var (
	flagProject string
	flagEnv     string

	// Shared across subcommands.
	cfg *config.Config
	api *client.Client
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "zenv",
		Short: "zEnv — zero-knowledge secret manager",
		Long:  "zEnv is a zero-knowledge encrypted vault for storing and sharing sensitive data.\nEven we as the provider cannot read your data.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg = config.Load(flagProject, flagEnv)
			if cfg.Token != "" {
				api = client.New(cfg.APIURL, cfg.Token)
			}
			return nil
		},
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&flagProject, "project", "", "project name or ID")
	root.PersistentFlags().StringVar(&flagEnv, "env", "", "environment: development, staging, production")

	root.AddCommand(newSecretsCmd())
	root.AddCommand(newRunCmd())
	root.AddCommand(newTokensCmd())
	root.AddCommand(newEnvCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newLoginCmd())
	root.AddCommand(newWhoamiCmd())

	return root
}

func requireConfig() error {
	if cfg.Token == "" {
		return fmt.Errorf("ZENV_TOKEN is not set.\nSet it: export ZENV_TOKEN=svc_...")
	}
	if cfg.VaultKey == "" {
		return fmt.Errorf("ZENV_VAULT_KEY is not set.\nSet it: export ZENV_VAULT_KEY=...")
	}
	if cfg.Project == "" {
		return fmt.Errorf("no project specified.\nUse --project, .zenv file, or ZENV_PROJECT")
	}
	if cfg.Env == "" {
		return fmt.Errorf("no environment specified.\nUse --env, .zenv file, or ZENV_ENV")
	}
	return nil
}

// getDEKAndHMACKey derives encryption keys from ZENV_VAULT_KEY.
//
// Phase 1 simplified flow: Argon2id(ZENV_VAULT_KEY, fixed salt) → DEK + HMAC key.
//
// Full flow (TODO): fetch project_salt from API → Argon2id → Project KEK → unwrap Project DEK.
func getDEKAndHMACKey() (dek, hmacKey []byte) {
	salt := []byte("zenv-phase1-project-salt-0000000") // 32 bytes, temporary
	dek, hmacKey = amnesia.DeriveKeys(cfg.VaultKey, salt, amnesia.KeyTypePassphrase)
	return dek, hmacKey
}
