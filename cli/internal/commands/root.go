package commands

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/cli/internal/client"
	"github.com/Judeadeniji/zenv-sh/cli/internal/config"
)

var (
	flagProject string
	flagEnv     string
	flagVerbose bool

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
			// Set log level.
			logLevel := slog.LevelWarn
			if flagVerbose {
				logLevel = slog.LevelDebug
			}
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

			cfg = config.Load(flagProject, flagEnv)
			slog.Debug("config loaded", "project", cfg.Project, "env", cfg.Env, "api", cfg.APIURL)

			if cfg.Token != "" {
				api = client.New(cfg.APIURL, cfg.Token)
			}
			return nil
		},
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&flagProject, "project", "", "project name or ID")
	root.PersistentFlags().StringVar(&flagEnv, "env", "", "environment: development, staging, production")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "enable debug logging")

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
// Flow (matches master plan Section 2.4.3):
// 1. GET /sdk/projects/{id}/crypto → project_salt + wrapped_project_dek
// 2. Argon2id(ZENV_VAULT_KEY + project_salt) → Project KEK
// 3. AES-256-GCM unwrap(wrapped_dek, Project KEK) → Project DEK
// 4. Project DEK used for encrypt/decrypt + as HMAC key for name hashing
func getDEKAndHMACKey() (dek, hmacKey []byte, err error) {
	// Fetch project crypto from API
	pc, err := api.GetProjectCrypto(cfg.Project)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch project crypto: %w", err)
	}

	// Decode base64 fields
	projectSalt, err := base64Decode(pc.ProjectSalt)
	if err != nil {
		return nil, nil, fmt.Errorf("decode project_salt: %w", err)
	}
	wrappedProjectDEK, err := base64Decode(pc.WrappedProjectDEK)
	if err != nil {
		return nil, nil, fmt.Errorf("decode wrapped_project_dek: %w", err)
	}

	// Derive Project KEK from ZENV_VAULT_KEY + project salt
	projectKEK, _ := amnesia.DeriveKeys(cfg.VaultKey, projectSalt, amnesia.KeyTypePassphrase)

	// Unwrap Project DEK: first 12 bytes = nonce, rest = ciphertext
	if len(wrappedProjectDEK) < 13 {
		return nil, nil, fmt.Errorf("wrapped project DEK too short")
	}
	nonce := wrappedProjectDEK[:12]
	ciphertext := wrappedProjectDEK[12:]

	projectDEK, err := amnesia.UnwrapKey(ciphertext, nonce, projectKEK)
	if err != nil {
		return nil, nil, fmt.Errorf("unwrap project DEK (wrong ZENV_VAULT_KEY?): %w", err)
	}

	// DEK used for encrypt/decrypt, also as HMAC key for name hashing
	return projectDEK, projectDEK, nil
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
