package commands

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `Manage global and local CLI configuration.

Global config:   ~/.config/zenv/config       (api_url, auth_url)
Credentials:     ~/.config/zenv/credentials  (token, vault_key)
Local config:    .zenv                       (project, env)

Keys: api_url, auth_url, token, vault_key, project, env`,
	}

	var flagLocal bool

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			if flagLocal {
				return config.SetLocal(key, value)
			}
			return config.Set(key, value)
		},
	}
	setCmd.Flags().BoolVar(&flagLocal, "local", false, "write to .zenv instead of global config")

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val := config.Get(args[0])
			if val == "" {
				return fmt.Errorf("key %q is not set", args[0])
			}
			fmt.Println(val)
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all config values",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			kv := config.ListGlobal()
			if len(kv) == 0 {
				fmt.Fprintln(os.Stderr, "No global config set.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			keys := make([]string, 0, len(kv))
			for k := range kv {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				v := kv[k]
				if config.IsSecret(k) {
					if len(v) > 12 {
						v = v[:8] + "..." + v[len(v)-4:]
					} else {
						v = "****"
					}
				}
				fmt.Fprintf(w, "%s\t%s\n", k, v)
			}
			return w.Flush()
		},
	}

	unsetCmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.Unset(args[0])
		},
	}

	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Print config directory path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.Dir())
		},
	}

	cmd.AddCommand(setCmd, getCmd, listCmd, unsetCmd, pathCmd)
	return cmd
}
