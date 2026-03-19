package commands

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/cli/internal/crypto"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "run -- COMMAND [ARGS...]",
		Short:              "Inject secrets as env vars and run a command",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Strip leading "--" if present.
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}
			if len(args) == 0 {
				return fmt.Errorf("no command specified.\nUsage: zenv run -- node server.js")
			}

			if err := requireConfig(); err != nil {
				return err
			}

			dek, _, err := getDEKAndHMACKey()
			if err != nil {
				return fmt.Errorf("key derivation failed: %w", err)
			}

			// Fetch all secrets for this project+environment.
			items, err := api.ListSecrets(cfg.Project, cfg.Env)
			if err != nil {
				return fmt.Errorf("list secrets: %w", err)
			}

			if len(items) == 0 {
				fmt.Fprintln(os.Stderr, "zenv: no secrets found, running command without injection")
			}

			// Bulk fetch all ciphertexts.
			hashes := make([]string, 0, len(items))
			for _, item := range items {
				hashes = append(hashes, item.NameHash)
			}

			secrets, err := api.BulkFetch(cfg.Project, cfg.Env, hashes)
			if err != nil {
				return fmt.Errorf("fetch secrets: %w", err)
			}

			// Decrypt all and build env.
			env := os.Environ()
			for _, s := range secrets {
				payload, err := crypto.DecryptSecret(s.Ciphertext, s.Nonce, dek)
				if err != nil {
					fmt.Fprintf(os.Stderr, "zenv: warning: failed to decrypt item %s\n", s.ID)
					continue
				}
				env = append(env, payload.Name+"="+payload.Value)
			}

			fmt.Fprintf(os.Stderr, "zenv: injected %d secrets into environment\n", len(secrets))

			// Exec — replace the current process.
			binary, err := exec.LookPath(args[0])
			if err != nil {
				return fmt.Errorf("command not found: %s", args[0])
			}

			return syscall.Exec(binary, args, env)
		},
	}
}
