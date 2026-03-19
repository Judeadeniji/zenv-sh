package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with zEnv — opens browser for OAuth",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("zenv login: not yet implemented")
			return nil
		},
	}
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current auth context and active project",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("zenv whoami: not yet implemented")
			return nil
		},
	}
}
