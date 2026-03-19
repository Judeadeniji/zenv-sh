package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Environment operations",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "pull",
			Short: "Write secrets to a local .env file",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("zenv env pull: not yet implemented")
				return nil
			},
		},
		&cobra.Command{
			Use:   "diff ENV1 ENV2",
			Short: "Compare secrets across two environments",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("zenv env diff %s %s: not yet implemented\n", args[0], args[1])
				return nil
			},
		},
	)

	return cmd
}

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate all required secrets exist — use in CI before deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("zenv check: not yet implemented")
			return nil
		},
	}
}
