package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Manage service tokens",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "create",
			Short: "Create a scoped service token",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("zenv tokens create: not yet implemented")
				return nil
			},
		},
		&cobra.Command{
			Use:   "revoke TOKEN_ID",
			Short: "Revoke a service token instantly",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("zenv tokens revoke %s: not yet implemented\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List active service tokens",
			Aliases: []string{"ls"},
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("zenv tokens list: not yet implemented")
				return nil
			},
		},
	)

	return cmd
}
