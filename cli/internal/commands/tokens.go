package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/cli/internal/client"
)

var (
	tokenName       string
	tokenPermission string
	tokenExpiresAt  string
)

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Manage service tokens",
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a scoped service token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			if tokenName == "" {
				return fmt.Errorf("--name is required")
			}
			if tokenPermission == "" {
				tokenPermission = "read"
			}

			req := client.TokenCreateRequest{
				ProjectID:   cfg.Project,
				Name:        tokenName,
				Environment: cfg.Env,
				Permission:  tokenPermission,
			}
			if tokenExpiresAt != "" {
				req.ExpiresAt = &tokenExpiresAt
			}

			t, err := api.CreateToken(req)
			if err != nil {
				return err
			}

			fmt.Println("Token created successfully.")
			fmt.Println("")
			fmt.Printf("  Token:       %s\n", t.Token)
			fmt.Printf("  ID:          %s\n", t.ID)
			fmt.Printf("  Name:        %s\n", t.Name)
			fmt.Printf("  Environment: %s\n", t.Environment)
			fmt.Printf("  Permission:  %s\n", t.Permission)
			fmt.Println("")
			fmt.Println("  This token will never be shown again. Save it now.")
			return nil
		},
	}
	createCmd.Flags().StringVar(&tokenName, "name", "", "token name (required)")
	createCmd.Flags().StringVar(&tokenPermission, "permission", "read", "read or read_write")
	createCmd.Flags().StringVar(&tokenExpiresAt, "expires-at", "", "optional expiry (RFC3339)")

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List active service tokens",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			tokens, err := api.ListTokens(cfg.Project)
			if err != nil {
				return err
			}

			if len(tokens) == 0 {
				fmt.Println("No tokens found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tENV\tPERMISSION\tCREATED")
			for _, t := range tokens {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					t.ID[:8]+"...", t.Name, t.Environment, t.Permission, t.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}

	revokeCmd := &cobra.Command{
		Use:   "revoke TOKEN_ID",
		Short: "Revoke a service token instantly",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}

			if err := api.RevokeToken(args[0]); err != nil {
				return err
			}
			fmt.Printf("Token %s revoked.\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(createCmd, listCmd, revokeCmd)
	return cmd
}
