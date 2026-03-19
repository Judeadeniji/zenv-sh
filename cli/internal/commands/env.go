package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/cli/internal/crypto"
)

var envPullOutput string

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Environment operations",
	}

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Write secrets to a local .env file",
		Long: `Fetch all secrets for the current project+environment, decrypt them
locally, and write to a .env file. Replaces the entire file content.

  zenv env pull --env development
  zenv env pull --env production -o .env.production`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			dek, hmacKey, err := getDEKAndHMACKey()
			if err != nil {
				return fmt.Errorf("key derivation failed: %w", err)
			}

			// List all secrets
			items, err := api.ListSecrets(cfg.Project, cfg.Env)
			if err != nil {
				return fmt.Errorf("list secrets: %w", err)
			}

			if len(items) == 0 {
				fmt.Fprintf(os.Stderr, "No secrets found in %s/%s\n", cfg.Project, cfg.Env)
				return nil
			}

			// Bulk fetch all ciphertext
			hashes := make([]string, 0, len(items))
			for _, item := range items {
				hashes = append(hashes, item.NameHash)
			}

			secrets, err := api.BulkFetch(cfg.Project, cfg.Env, hashes)
			if err != nil {
				return fmt.Errorf("fetch secrets: %w", err)
			}

			// Decrypt and collect key=value pairs
			var lines []string
			_ = hmacKey // only needed for hashing names, not decryption
			for _, s := range secrets {
				payload, err := crypto.DecryptSecret(s.Ciphertext, s.Nonce, dek)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to decrypt %s: %v\n", s.NameHash[:8], err)
					continue
				}
				lines = append(lines, fmt.Sprintf("%s=%s", payload.Name, payload.Value))
			}

			sort.Strings(lines)

			// Write to file or stdout
			output := envPullOutput
			if output == "" {
				output = ".env.local"
			}

			if output == "-" {
				for _, line := range lines {
					fmt.Println(line)
				}
			} else {
				content := ""
				for _, line := range lines {
					content += line + "\n"
				}
				if err := os.WriteFile(output, []byte(content), 0600); err != nil {
					return fmt.Errorf("write %s: %w", output, err)
				}
				fmt.Fprintf(os.Stderr, "Wrote %d secrets to %s\n", len(lines), output)
			}
			return nil
		},
	}
	pullCmd.Flags().StringVarP(&envPullOutput, "output", "o", "", "output file (default: .env.local, use - for stdout)")

	diffCmd := &cobra.Command{
		Use:   "diff ENV1 ENV2",
		Short: "Compare secrets across two environments",
		Long: `Show which secrets exist in each environment and highlight differences.

  zenv env diff development production`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			env1, env2 := args[0], args[1]

			// List secrets in both environments
			items1, err := api.ListSecrets(cfg.Project, env1)
			if err != nil {
				return fmt.Errorf("list %s: %w", env1, err)
			}
			items2, err := api.ListSecrets(cfg.Project, env2)
			if err != nil {
				return fmt.Errorf("list %s: %w", env2, err)
			}

			// Build sets of name hashes
			set1 := make(map[string]bool, len(items1))
			for _, item := range items1 {
				set1[item.NameHash] = true
			}
			set2 := make(map[string]bool, len(items2))
			for _, item := range items2 {
				set2[item.NameHash] = true
			}

			// Collect all unique hashes
			allHashes := make(map[string]bool)
			for h := range set1 {
				allHashes[h] = true
			}
			for h := range set2 {
				allHashes[h] = true
			}

			// Report
			var onlyIn1, onlyIn2, inBoth int
			for h := range allHashes {
				in1 := set1[h]
				in2 := set2[h]
				hash := h
				if len(hash) > 12 {
					hash = hash[:12] + "..."
				}

				switch {
				case in1 && in2:
					fmt.Printf("  = %s  (both)\n", hash)
					inBoth++
				case in1 && !in2:
					fmt.Printf("  - %s  (only in %s)\n", hash, env1)
					onlyIn1++
				case !in1 && in2:
					fmt.Printf("  + %s  (only in %s)\n", hash, env2)
					onlyIn2++
				}
			}

			fmt.Fprintf(os.Stderr, "\n%d shared, %d only in %s, %d only in %s\n",
				inBoth, onlyIn1, env1, onlyIn2, env2)
			return nil
		},
	}

	cmd.AddCommand(pullCmd, diffCmd)
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

			_, hmacKey, err := getDEKAndHMACKey()
			if err != nil {
				return fmt.Errorf("key derivation failed: %w", err)
			}

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
