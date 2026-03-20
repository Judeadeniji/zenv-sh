package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with zEnv",
		Long: `Authenticate with zEnv by providing a service token.

You can get a service token from the zEnv dashboard or by running:
  zenv tokens create --name "cli" --project <project-id>

After login, the token is stored in ~/.config/zenv/credentials
and used automatically for subsequent commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			authURL := os.Getenv("ZENV_AUTH_URL")
			if authURL == "" {
				authURL = "http://localhost:3000"
			}

			fmt.Fprintln(os.Stderr, "Get a service token from your zEnv dashboard.")
			fmt.Fprintf(os.Stderr, "Dashboard: %s\n\n", authURL)

			// Prompt for token
			fmt.Fprint(os.Stderr, "Paste your service token: ")
			reader := bufio.NewReader(os.Stdin)
			token, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read token: %w", err)
			}
			token = strings.TrimSpace(token)

			if token == "" {
				return fmt.Errorf("no token provided")
			}

			if !strings.HasPrefix(token, "svc_") {
				return fmt.Errorf("invalid token format — service tokens start with svc_")
			}

			// Store to ~/.config/zenv/credentials
			configDir, err := os.UserConfigDir()
			if err != nil {
				return fmt.Errorf("resolve config dir: %w", err)
			}
			zenvDir := configDir + "/zenv"
			if err := os.MkdirAll(zenvDir, 0700); err != nil {
				return fmt.Errorf("create config dir: %w", err)
			}

			credPath := zenvDir + "/credentials"
			if err := os.WriteFile(credPath, []byte("ZENV_TOKEN="+token+"\n"), 0600); err != nil {
				return fmt.Errorf("write credentials: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Token saved to %s\n", credPath)
			fmt.Fprintln(os.Stderr, "You can also set ZENV_TOKEN in your shell to override.")
			return nil
		},
	}

	return cmd
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current auth context and active project",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "API:         %s\n", cfg.APIURL)

			if cfg.Token != "" {
				masked := cfg.Token[:min(8, len(cfg.Token))] + "..." + cfg.Token[max(0, len(cfg.Token)-4):]
				fmt.Fprintf(os.Stderr, "Token:       %s\n", masked)
			} else {
				fmt.Fprintln(os.Stderr, "Token:       (not set)")
			}

			if cfg.VaultKey != "" {
				fmt.Fprintln(os.Stderr, "Vault Key:   (set)")
			} else {
				fmt.Fprintln(os.Stderr, "Vault Key:   (not set)")
			}

			if cfg.Project != "" {
				fmt.Fprintf(os.Stderr, "Project:     %s\n", cfg.Project)
			} else {
				fmt.Fprintln(os.Stderr, "Project:     (not set)")
			}

			if cfg.Env != "" {
				fmt.Fprintf(os.Stderr, "Environment: %s\n", cfg.Env)
			} else {
				fmt.Fprintln(os.Stderr, "Environment: (not set)")
			}

			// Validate token against the API if we have enough context.
			if api != nil && cfg.Project != "" {
				_, err := api.GetProject(cfg.Project)
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nStatus:      invalid or expired (%s)\n", err)
				} else {
					fmt.Fprintln(os.Stderr, "\nStatus:      authenticated")
				}
			}

			return nil
		},
	}
}

// openBrowser opens a URL in the user's default browser.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
