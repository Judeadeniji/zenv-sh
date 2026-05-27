package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Judeadeniji/zenv-sh/cli/internal/crypto"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run -- COMMAND [ARGS...]",
		Short: "Inject secrets as env vars and run a command",
		RunE:  runCmd,
	}
	cmd.Flags().Bool("allow-partial", false, "Continue even if some secrets fail to decrypt")
	cmd.Flags().Bool("buildkit", false, "Inject secrets as Docker BuildKit --secret arguments")
	return cmd
}

func runCmd(cmd *cobra.Command, args []string) error {
	allowPartial, _ := cmd.Flags().GetBool("allow-partial")
	useBuildKit, _ := cmd.Flags().GetBool("buildkit")

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

	// Auto-detect Docker BuildKit usage
	if len(cmdArgs) > 1 && cmdArgs[0] == "docker" && cmdArgs[1] == "build" {
		useBuildKit = true
	} else if len(cmdArgs) > 2 && cmdArgs[0] == "docker" && cmdArgs[1] == "buildx" && cmdArgs[2] == "build" {
		useBuildKit = true
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

	if useBuildKit {
		var secretNames []string
		for name := range overrides {
			secretNames = append(secretNames, name)
		}
		cmdArgs = injectBuildKitSecrets(cmdArgs, secretNames)
	}

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

// injectBuildKitSecrets splices Docker BuildKit --secret arguments into the command array.
func injectBuildKitSecrets(args []string, keys []string) []string {
	if len(keys) == 0 {
		return args
	}

	var secretFlags []string
	for _, key := range keys {
		secretFlags = append(secretFlags, "--secret", fmt.Sprintf("id=%s,env=%s", key, key))
	}

	// Insert after `build` or `buildx build` to ensure valid Docker syntax.
	insertIdx := 1
	if len(args) > 1 && args[0] == "docker" && args[1] == "build" {
		insertIdx = 2
	} else if len(args) > 2 && args[0] == "docker" && args[1] == "buildx" && args[2] == "build" {
		insertIdx = 3
	} else if len(args) > 1 && args[1] == "build" {
		insertIdx = 2 // e.g., nerdctl build
	}

	result := make([]string, 0, len(args)+len(secretFlags))
	result = append(result, args[:insertIdx]...)
	result = append(result, secretFlags...)
	result = append(result, args[insertIdx:]...)
	return result
}
