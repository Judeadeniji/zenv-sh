package main

import (
	"os"

	"github.com/Judeadeniji/zenv-sh/cli/internal/commands"
)

func main() {
	root := commands.NewRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
