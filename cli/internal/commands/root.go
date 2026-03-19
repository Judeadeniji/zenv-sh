package commands

import (
	"github.com/spf13/cobra"
)

var (
	flagProject string
	flagEnv     string
)

// NewRootCmd creates the root zenv command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "zenv",
		Short: "zEnv — zero-knowledge secret manager",
		Long:  "zEnv is a zero-knowledge encrypted vault for storing and sharing sensitive data.\nEven we as the provider cannot read your data.",
	}

	// Global flags
	root.PersistentFlags().StringVar(&flagProject, "project", "", "project name (overrides .zenv config)")
	root.PersistentFlags().StringVar(&flagEnv, "env", "", "environment: development, staging, production")

	// Subcommands
	root.AddCommand(newLoginCmd())
	root.AddCommand(newWhoamiCmd())
	root.AddCommand(newSecretsCmd())
	root.AddCommand(newRunCmd())
	root.AddCommand(newTokensCmd())
	root.AddCommand(newEnvCmd())
	root.AddCommand(newCheckCmd())

	return root
}
