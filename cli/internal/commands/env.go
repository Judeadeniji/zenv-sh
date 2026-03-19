package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/cli/internal/crypto"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Environment operations",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "pull",
			Short: "Write secrets to a local .env file",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("zenv env pull: not yet implemented")
				return nil
			},
		},
		&cobra.Command{
			Use:   "diff ENV1 ENV2",
			Short: "Compare secrets across two environments",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("zenv env diff %s %s: not yet implemented\n", args[0], args[1])
				return nil
			},
		},
	)

	return cmd
}

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check KEY [KEY...]",
		Short: "Validate that required secrets exist — use in CI before deployment",
		Long: `Check that all specified secret keys exist in the vault for the current
project and environment. Exits with code 1 if any are missing.

Use in CI pipelines to fail fast before deployment:
  zenv check DATABASE_URL STRIPE_KEY JWT_SECRET --env production`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			_, hmacKey := getDEKAndHMACKey()

			// Build name hashes for all requested keys.
			hashes := make([]string, 0, len(args))
			hashToName := make(map[string]string, len(args))
			for _, name := range args {
				h := crypto.ComputeNameHash(name, hmacKey)
				hashes = append(hashes, h)
				hashToName[h] = name
			}

			// Bulk fetch from API.
			items, err := api.BulkFetch(cfg.Project, cfg.Env, hashes)
			if err != nil {
				return fmt.Errorf("check: %w", err)
			}

			// Build set of found hashes.
			found := make(map[string]bool, len(items))
			for _, item := range items {
				found[item.NameHash] = true
			}

			// Report results.
			missing := 0
			for _, h := range hashes {
				name := hashToName[h]
				if found[h] {
					fmt.Fprintf(os.Stdout, "  ✓ %s\n", name)
				} else {
					fmt.Fprintf(os.Stdout, "  ✗ %s — missing\n", name)
					missing++
				}
			}

			if missing > 0 {
				fmt.Fprintf(os.Stderr, "\nzenv check: %d of %d secrets missing in %s/%s\n", missing, len(args), cfg.Project, cfg.Env)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "\nzenv check: all %d secrets present in %s/%s\n", len(args), cfg.Project, cfg.Env)
			return nil
		},
	}

	return cmd
}
