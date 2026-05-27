package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Judeadeniji/zenv-sh/sdk-go/crypto"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run -- COMMAND [ARGS...]",
		Short: "Inject secrets as env vars and run a command",
		RunE:  runCmd,
	}
	cmd.Flags().Bool("allow-partial", false, "Continue even if some secrets fail to decrypt")
	return cmd
}

func runCmd(cmd *cobra.Command, args []string) error {
	allowPartial, _ := cmd.Flags().GetBool("allow-partial")

	// Require an explicit "--" separator so child command flags (e.g. node --inspect)
	// are never misinterpreted by cobra.
	dashIdx := cmd.ArgsLenAtDash()
	if dashIdx == -1 {
		return fmt.Errorf("missing '--' separator\nUsage: zenv run -- node server.js")
	}
	cmdArgs := args[dashIdx:]
	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified\nUsage: zenv run -- node server.js")
	}

	if err := requireConfig(); err != nil {
		return err
	}

	dek, hmacKey, err := getDEKAndHMACKey()
	if err != nil {
		return fmt.Errorf("key derivation failed: %w", err)
	}
	// NOTE: no defer — on Unix, syscall.Exec replaces the process image and
	// deferred calls never execute. Zero keys explicitly, before exec, below.

	items, err := api.ListSecrets(cfg.Project, cfg.Env)
	if err != nil {
		zeroBytes(dek)
		zeroBytes(hmacKey)
		return fmt.Errorf("list secrets: %w", err)
	}
	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "zenv: no secrets found, running command without injection")
	}

	hashes := make([]string, 0, len(items))
	for _, item := range items {
		hashes = append(hashes, item.NameHash)
	}

	secrets, err := api.BulkFetch(cfg.Project, cfg.Env, hashes)
	if err != nil {
		zeroBytes(dek)
		zeroBytes(hmacKey)
		return fmt.Errorf("fetch secrets: %w", err)
	}

	// Decrypt into a map so mergeEnv can deduplicate against os.Environ().
	overrides := make(map[string]string, len(secrets))
	for _, s := range secrets {
		payload, err := crypto.DecryptSecret(s.Ciphertext, s.Nonce, dek)
		if err != nil {
			if !allowPartial {
				zeroBytes(dek)
				zeroBytes(hmacKey)
				return fmt.Errorf("failed to decrypt secret %s (use --allow-partial to skip): %w", s.ID, err)
			}
			fmt.Fprintf(os.Stderr, "zenv: warning: skipping secret %s: %v\n", s.ID, err)
			continue
		}
		overrides[payload.Name] = payload.Value
	}

	injected := len(overrides)
	env := mergeEnv(os.Environ(), overrides)

	// All decryption is done. Zero key material explicitly before handing off
	// to the child process — defers are not reliable here (see note above).
	zeroBytes(dek)
	zeroBytes(hmacKey)

	fmt.Fprintf(os.Stderr, "zenv: injected %d/%d secrets\n", injected, len(secrets))

	binary, err := exec.LookPath(cmdArgs[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", cmdArgs[0])
	}

	return execCommand(binary, cmdArgs, env)
}

// mergeEnv builds an env slice from base, with overrides winning on collision.
// This prevents duplicate keys that cause undefined behaviour in child processes.
func mergeEnv(base []string, overrides map[string]string) []string {
	result := make([]string, 0, len(base)+len(overrides))
	for _, kv := range base {
		idx := strings.IndexByte(kv, '=')
		if idx == -1 {
			// Malformed entry — keep as-is.
			result = append(result, kv)
			continue
		}
		if _, overridden := overrides[kv[:idx]]; !overridden {
			result = append(result, kv)
		}
	}
	for name, value := range overrides {
		result = append(result, name+"="+value)
	}
	return result
}

// zeroBytes overwrites a byte slice with zeros.
// Go's GC does not zero heap memory on collection, so this must be called
// explicitly for any slice holding key material.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
