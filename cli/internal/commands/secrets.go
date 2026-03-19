package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/cli/internal/crypto"
)

func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "secrets",
		Aliases: []string{"s"},
		Short:   "Manage secrets",
	}

	cmd.AddCommand(
		newSecretsGetCmd(),
		newSecretsSetCmd(),
		newSecretsListCmd(),
		newSecretsDeleteCmd(),
	)

	return cmd
}

func newSecretsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get KEY",
		Short: "Retrieve and decrypt a secret value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			name := args[0]
			dek, hmacKey := getDEKAndHMACKey()
			nameHash := crypto.NameHashURL(name, hmacKey)

			item, err := api.GetSecret(cfg.Project, cfg.Env, nameHash)
			if err != nil {
				return fmt.Errorf("get secret: %w", err)
			}

			payload, err := crypto.DecryptSecret(item.Ciphertext, item.Nonce, dek)
			if err != nil {
				return fmt.Errorf("decrypt: %w", err)
			}

			fmt.Print(payload.Value)
			return nil
		},
	}
}

func newSecretsSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Create or update an encrypted secret",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			name, value := args[0], args[1]
			dek, hmacKey := getDEKAndHMACKey()

			ct, nonce, nameHash, err := crypto.EncryptSecret(name, value, dek, hmacKey)
			if err != nil {
				return fmt.Errorf("encrypt: %w", err)
			}

			// Try create first, if conflict then update.
			_, err = api.CreateSecret(cfg.Project, cfg.Env, nameHash, ct, nonce)
			if err != nil {
				// Likely already exists — try update.
				nameHashURL := crypto.NameHashURL(name, hmacKey)
				_, err = api.UpdateSecret(cfg.Project, cfg.Env, nameHashURL, ct, nonce)
				if err != nil {
					return fmt.Errorf("set secret: %w", err)
				}
				fmt.Fprintf(os.Stderr, "updated %s\n", name)
				return nil
			}

			fmt.Fprintf(os.Stderr, "created %s\n", name)
			return nil
		},
	}
}

func newSecretsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List secrets (names are hashed — shows metadata only)",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			items, err := api.ListSecrets(cfg.Project, cfg.Env)
			if err != nil {
				return fmt.Errorf("list secrets: %w", err)
			}

			if len(items) == 0 {
				fmt.Println("no secrets found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME_HASH\tVERSION\tUPDATED")
			for _, item := range items {
				fmt.Fprintf(w, "%s\t%d\t%s\n", item.NameHash[:16]+"...", item.Version, item.UpdatedAt)
			}
			w.Flush()
			return nil
		},
	}
}

func newSecretsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete KEY",
		Short:   "Delete a secret",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			name := args[0]
			_, hmacKey := getDEKAndHMACKey()
			nameHash := crypto.NameHashURL(name, hmacKey)

			if err := api.DeleteSecret(cfg.Project, cfg.Env, nameHash); err != nil {
				return fmt.Errorf("delete secret: %w", err)
			}

			fmt.Fprintf(os.Stderr, "deleted %s\n", name)
			return nil
		},
	}
}
