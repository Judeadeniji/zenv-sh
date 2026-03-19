package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run -- COMMAND [ARGS...]",
		Short: "Inject secrets as env vars and run a command",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("zenv run: not yet implemented")
			return nil
		},
	}
}
