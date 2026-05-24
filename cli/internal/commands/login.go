package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/cli/internal/client"
	"github.com/Judeadeniji/zenv-sh/cli/internal/config"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Save a service token for API authentication",
		Long: `Authenticate with zEnv using a service token (for CI, bots, and automation).

Create a token in the zEnv dashboard, then paste it here. Only the token is
stored — no Vault Key prompt. Service tokens are meant for unattended use.

Credentials are stored per project under ~/.config/zenv/projects/<id>/.

To decrypt secrets you also need the project vault key:
  • CI / machines: export ZENV_PROJECT_KEY=<from dashboard>
  • Developer laptop: zenv unlock (saves key for that project only)

Run "zenv whoami" to verify.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "Get a service token from your zEnv dashboard.")
			fmt.Fprintf(os.Stderr, "Dashboard: %s\n\n", cfg.AuthURL)

			fmt.Fprint(os.Stderr, "Paste your service token: ")
			token, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read token: %w", err)
			}
			token = strings.TrimSpace(token)

			if token == "" {
				return fmt.Errorf("no token provided")
			}
			if !strings.HasPrefix(token, "ze_") {
				return fmt.Errorf("invalid token format — service tokens start with ze_")
			}

			apiClient := client.New(cfg.APIURL, token)
			projectID, err := saveTokenForProject(apiClient, token)
			if err != nil {
				return err
			}

			if err := syncScopeFromToken(apiClient, projectID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not sync project scope: %s\n", err)
			}

			printProjectKeyHint(projectID)

			fmt.Fprintln(os.Stderr, "\nRun `zenv whoami` to verify.")
			return nil
		},
	}
}

// saveTokenForProject stores the token in the project-scoped credentials file.
func saveTokenForProject(apiClient *client.Client, token string) (string, error) {
	info, err := apiClient.Whoami()
	if err != nil {
		return "", fmt.Errorf("verify token: %w", err)
	}
	if info.ProjectID == "" {
		return "", fmt.Errorf("token has no project scope — cannot determine where to save credentials")
	}
	if err := config.SetForProject(info.ProjectID, config.KeyToken, token); err != nil {
		return "", fmt.Errorf("save token: %w", err)
	}
	credPath, _ := config.ProjectCredentialsPath(info.ProjectID)
	fmt.Fprintf(os.Stderr, "Token saved for project %s:\n  %s\n", info.ProjectID, credPath)
	return info.ProjectID, nil
}

// syncScopeFromToken writes project/env from the token into .zenv.
func syncScopeFromToken(apiClient *client.Client, projectID string) error {
	if projectID == "" {
		info, err := apiClient.Whoami()
		if err != nil {
			return err
		}
		projectID = info.ProjectID
	}
	if projectID == "" {
		return nil
	}
	if err := config.SetLocal(config.KeyProject, projectID); err != nil {
		return fmt.Errorf("save project to .zenv: %w", err)
	}
	info, _ := apiClient.Whoami()
	if info != nil && info.Environment != "" {
		if err := config.SetLocal(config.KeyEnv, info.Environment); err != nil {
			return fmt.Errorf("save env to .zenv: %w", err)
		}
	}
	fmt.Fprintf(os.Stderr, "Linked .zenv to project %s", projectID)
	if info != nil && info.Environment != "" {
		fmt.Fprintf(os.Stderr, " (%s)", info.Environment)
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

func printProjectKeyHint(projectID string) {
	if projectID != "" && config.GetForProject(projectID, config.KeyProjectKey) != "" {
		return
	}
	if os.Getenv("ZENV_PROJECT_KEY") != "" {
		return
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Project key not configured (required to encrypt/decrypt secrets).")
	fmt.Fprintln(os.Stderr, "  CI / automation:  export ZENV_PROJECT_KEY=<from dashboard>")
	fmt.Fprintln(os.Stderr, "  Developer setup:  zenv unlock")
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show your identity, token, and project scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				fmt.Fprintln(os.Stderr, "Not authenticated.")
				fmt.Fprintln(os.Stderr, "Run: zenv login")
				return nil
			}

			info, err := api.Whoami()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Authentication failed: %s\n", err)
				fmt.Fprintln(os.Stderr, "Run: zenv login")
				return nil
			}

			if info.UserName != "" {
				if info.UserEmail != "" {
					fmt.Fprintf(os.Stderr, "Logged in as %s <%s>\n\n", info.UserName, info.UserEmail)
				} else {
					fmt.Fprintf(os.Stderr, "Logged in as %s\n\n", info.UserName)
				}
			} else if info.UserEmail != "" {
				fmt.Fprintf(os.Stderr, "Logged in as %s\n\n", info.UserEmail)
			}

			fmt.Fprintf(os.Stderr, "Token:       %s\n", info.TokenName)
			if info.OrganizationName != "" {
				fmt.Fprintf(os.Stderr, "Org:         %s\n", info.OrganizationName)
			} else if info.OrganizationID != "" {
				fmt.Fprintf(os.Stderr, "Org:         %s\n", info.OrganizationID)
			}

			if cfg.Project != "" {
				if p, err := api.GetProject(cfg.Project); err == nil && p.Name != "" {
					fmt.Fprintf(os.Stderr, "Project:     %s (%s)\n", p.Name, cfg.Project)
				} else {
					fmt.Fprintf(os.Stderr, "Project:     %s\n", cfg.Project)
				}
			} else if info.ProjectName != "" {
				fmt.Fprintf(os.Stderr, "Project:     %s\n", info.ProjectName)
			} else {
				fmt.Fprintf(os.Stderr, "Project:     %s\n", info.ProjectID)
			}

			env := cfg.Env
			if env == "" {
				env = info.Environment
			}
			fmt.Fprintf(os.Stderr, "Environment: %s\n", env)
			fmt.Fprintf(os.Stderr, "Permission:  %s\n", info.Permission)

			if cfg.ProjectKey != "" {
				fmt.Fprintln(os.Stderr, "Project Key: set")
				if cfg.Project != "" {
					if path, err := config.ProjectCredentialsPath(cfg.Project); err == nil {
						fmt.Fprintf(os.Stderr, "Credentials: %s\n", path)
					}
				}
			} else {
				fmt.Fprintln(os.Stderr, "Project Key: (not set)")
				fmt.Fprintln(os.Stderr, "  Set ZENV_PROJECT_KEY or run: zenv unlock")
			}

			if projects, err := config.ListConfiguredProjects(); err == nil && len(projects) > 1 {
				fmt.Fprintf(os.Stderr, "\nOther configured projects: %d (zenv config profiles)\n", len(projects))
			}

			if info.ProjectID != "" && cfg.Project != "" && info.ProjectID != cfg.Project {
				fmt.Fprintf(os.Stderr,
					"\nWarning: token is scoped to project %s but CLI is using %s.\n",
					info.ProjectID, cfg.Project,
				)
				fmt.Fprintf(os.Stderr, "  Run: zenv projects init %s\n", info.ProjectID)
			}
			if info.Environment != "" && cfg.Env != "" && info.Environment != cfg.Env {
				fmt.Fprintf(os.Stderr,
					"\nWarning: token is scoped to env %s but CLI is using %s.\n",
					info.Environment, cfg.Env,
				)
			}

			return nil
		},
	}
}
