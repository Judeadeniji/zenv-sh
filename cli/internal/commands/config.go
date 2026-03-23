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
		Long: `Manage CLI configuration (like git config).

By default, reads/writes the local .zenv file.
Use --global to target ~/.config/zenv/ instead.

  zenv config set project <uuid>           # writes to .zenv
  zenv config set env production           # writes to .zenv
  zenv config set --global api_url http://zenv.localhost
  zenv config set --global token ze_...   # stored in credentials (0600)

Keys: api_url, auth_url, token, project_key, project, env`,
	}

	var flagGlobal bool

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagGlobal {
				return config.Set(args[0], args[1])
			}
			return config.SetLocal(args[0], args[1])
		},
	}
	setCmd.Flags().BoolVar(&flagGlobal, "global", false, "write to global config (~/.config/zenv/)")

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var val string
			if flagGlobal {
				val = config.Get(args[0])
			} else {
				val = config.GetLocal(args[0])
			}
			if val == "" {
				return fmt.Errorf("key %q is not set", args[0])
			}
			fmt.Println(val)
			return nil
		},
	}
	getCmd.Flags().BoolVar(&flagGlobal, "global", false, "read from global config")

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List all config values",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			if flagGlobal {
				printKV(w, config.ListGlobal())
			} else {
				// Show both scopes
				globalKV := config.ListGlobal()
				localKV := config.ListLocal()
				if len(globalKV) == 0 && len(localKV) == 0 {
					fmt.Fprintln(os.Stderr, "No config set.")
					return nil
				}
				if len(globalKV) > 0 {
					fmt.Fprintln(w, "# global (~/.config/zenv/)")
					printKV(w, globalKV)
				}
				if len(localKV) > 0 {
					if len(globalKV) > 0 {
						fmt.Fprintln(w)
					}
					fmt.Fprintln(w, "# local (.zenv)")
					printKV(w, localKV)
				}
			}

			return w.Flush()
		},
	}
	listCmd.Flags().BoolVar(&flagGlobal, "global", false, "list only global config")

	unsetCmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagGlobal {
				return config.Unset(args[0])
			}
			return config.UnsetLocal(args[0])
		},
	}
	unsetCmd.Flags().BoolVar(&flagGlobal, "global", false, "remove from global config")

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

func printKV(w *tabwriter.Writer, kv map[string]string) {
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
		fmt.Fprintf(w, "  %s\t%s\n", k, v)
	}
}
