package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "secrets",
		Aliases: []string{"s"},
		Short:   "Manage secrets",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "get KEY",
			Short: "Retrieve a secret value",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("zenv secrets get %s: not yet implemented\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "set KEY VALUE",
			Short: "Create or update a secret",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("zenv secrets set %s: not yet implemented\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List secret names (never values)",
			Aliases: []string{"ls"},
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("zenv secrets list: not yet implemented")
				return nil
			},
		},
		&cobra.Command{
			Use:   "delete KEY",
			Short: "Delete a secret",
			Aliases: []string{"rm"},
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("zenv secrets delete %s: not yet implemented\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "versions KEY",
			Short: "Show version history for a secret",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("zenv secrets versions %s: not yet implemented\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "rollback KEY",
			Short: "Revert to a previous version",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("zenv secrets rollback %s: not yet implemented\n", args[0])
				return nil
			},
		},
	)

	return cmd
}
