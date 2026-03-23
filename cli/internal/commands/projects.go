package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/cli/internal/client"
	"github.com/Judeadeniji/zenv-sh/cli/internal/config"
)

var (
	projectOrgID  string
	projectName   string
	projectPubKey string
)

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"p"},
		Short:   "Manage projects",
	}

	cmd.AddCommand(
		newProjectsInitCmd(),
		newProjectsListCmd(),
		newProjectsGetCmd(),
		newProjectsCreateCmd(),
	)

	return cmd
}

func newProjectsInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init PROJECT_ID",
		Short: "Write a .zenv file to pin this directory to a project",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing PROJECT_ID.\n\nUsage: zenv projects init <project-id>\n\nFind your project ID with: zenv projects list")
			}
			if len(args) > 1 {
				return fmt.Errorf("expected 1 argument, got %d", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			if err := config.SetLocal("project", projectID); err != nil {
				return fmt.Errorf("write .zenv: %w", err)
			}
			if flagEnv != "" {
				if err := config.SetLocal("env", flagEnv); err != nil {
					return fmt.Errorf("write .zenv: %w", err)
				}
			}

			fmt.Fprintf(os.Stderr, "updated .zenv (project=%s", projectID)
			if flagEnv != "" {
				fmt.Fprintf(os.Stderr, ", env=%s", flagEnv)
			}
			fmt.Fprintln(os.Stderr, ")")
			return nil
		},
	}
}

func newProjectsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List projects in an organization",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				return fmt.Errorf("not authenticated.\nRun: zenv login\n  or: zenv config set --global token <your-service-token>")
			}

			orgID := resolveOrgID()
			if orgID == "" {
				info, err := api.Whoami()
				if err == nil && info.OrganizationID != "" {
					orgID = info.OrganizationID
				}
			}
			if orgID == "" {
				return fmt.Errorf("--org is required (or set ZENV_ORGANIZATION)")
			}

			projects, err := api.ListProjects(orgID)
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				fmt.Println("No projects found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tCREATED")
			for _, p := range projects {
				fmt.Fprintf(w, "%s\t%s\t%s\n", p.ID, p.Name, p.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&projectOrgID, "org", "", "organization ID")
	return cmd
}

func newProjectsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get PROJECT_ID",
		Short: "Get project details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				return fmt.Errorf("not authenticated.\nRun: zenv login\n  or: zenv config set --global token <your-service-token>")
			}

			p, err := api.GetProject(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("  ID:           %s\n", p.ID)
			fmt.Printf("  Name:         %s\n", p.Name)
			fmt.Printf("  Organization: %s\n", p.OrganizationID)
			fmt.Printf("  Created:      %s\n", p.CreatedAt)
			return nil
		},
	}
}

func newProjectsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a project with client-side crypto",
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				return fmt.Errorf("not authenticated.\nRun: zenv login\n  or: zenv config set --global token <your-service-token>")
			}

			orgID := resolveOrgID()
			if orgID == "" {
				info, err := api.Whoami()
				if err == nil && info.OrganizationID != "" {
					orgID = info.OrganizationID
				}
			}
			if orgID == "" {
				return fmt.Errorf("--org is required (or set ZENV_ORGANIZATION)")
			}
			if projectName == "" {
				return fmt.Errorf("--name is required")
			}
			if cfg.ProjectKey == "" {
				return fmt.Errorf("ZENV_PROJECT_KEY is not set.\nSet it: export ZENV_PROJECT_KEY=...")
			}
			if projectPubKey == "" {
				return fmt.Errorf("--public-key is required (base64 X25519 public key from vault unlock)")
			}

			pubKeyBytes, err := base64.StdEncoding.DecodeString(projectPubKey)
			if err != nil {
				return fmt.Errorf("invalid base64 in --public-key: %w", err)
			}

			// Generate project crypto (all client-side).
			projectSalt := amnesia.GenerateSalt()
			projectDEK := amnesia.GenerateKey()

			// Derive Project KEK from ZENV_PROJECT_KEY + project salt.
			projectKEK, _ := amnesia.DeriveKeys(cfg.ProjectKey, projectSalt, amnesia.KeyTypePassphrase)

			// Wrap Project DEK with Project KEK (nonce || ciphertext).
			wrappedCT, wrappedNonce, err := amnesia.WrapKey(projectDEK, projectKEK)
			if err != nil {
				return fmt.Errorf("wrap project DEK: %w", err)
			}
			wrappedProjectDEK := append(wrappedNonce, wrappedCT...)

			// Wrap Project DEK with user's public key (for key grant / team sharing).
			wrappedProjectVaultKey, err := amnesia.WrapWithPublicKey(projectDEK, pubKeyBytes)
			if err != nil {
				return fmt.Errorf("wrap project vault key: %w", err)
			}

			p, err := api.CreateProject(client.CreateProjectRequest{
				OrganizationID:         orgID,
				Name:                   projectName,
				ProjectSalt:            base64.StdEncoding.EncodeToString(projectSalt),
				WrappedProjectDEK:      base64.StdEncoding.EncodeToString(wrappedProjectDEK),
				WrappedProjectVaultKey: base64.StdEncoding.EncodeToString(wrappedProjectVaultKey),
			})
			if err != nil {
				return err
			}

			fmt.Println("Project created successfully.")
			fmt.Println("")
			fmt.Printf("  ID:   %s\n", p.ID)
			fmt.Printf("  Name: %s\n", p.Name)
			fmt.Println("")
			fmt.Printf("  Run 'zenv projects init %s' to link this directory.\n", p.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&projectOrgID, "org", "", "organization ID")
	cmd.Flags().StringVar(&projectName, "name", "", "project name (required)")
	cmd.Flags().StringVar(&projectPubKey, "public-key", "", "base64 X25519 public key (from vault unlock)")
	return cmd
}

func resolveOrgID() string {
	if projectOrgID != "" {
		return projectOrgID
	}
	return os.Getenv("ZENV_ORGANIZATION")
}
