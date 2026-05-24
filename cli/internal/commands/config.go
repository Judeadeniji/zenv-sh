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

Layout:
  ~/.config/zenv/config                         api_url, auth_url
  ~/.config/zenv/projects/<project-id>/credentials   token, project_key (per project)
  .zenv in your repo                            project, env

  zenv config set project <uuid>       # .zenv — which project this repo uses
  zenv config set env production       # .zenv
  zenv config set token ze_...         # current project's credentials (from .zenv)
  zenv config set --global api_url …   # install-wide settings only

  zenv config profiles                 # list projects with saved credentials`,
	}

	var flagGlobal bool
	var flagProjectID string

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			if config.IsLocalOnly(key) {
				if flagGlobal {
					return fmt.Errorf("%s is per-repo — omit --global (writes .zenv)", key)
				}
				return config.SetLocal(key, value)
			}

			if config.IsSecret(key) {
				if flagGlobal {
					fmt.Fprintln(os.Stderr, "Warning: storing secrets globally shares them across every project.")
					return config.SetGlobalSecret(key, value)
				}
				projectID := flagProjectID
				if projectID == "" {
					projectID = config.ResolveProject(flagProject)
				}
				if projectID == "" {
					return fmt.Errorf("no project context — run: zenv projects init <project-id>\n  or: zenv config set --project <id> %s <value>", key)
				}
				return config.SetForProject(projectID, key, value)
			}

			if !flagGlobal {
				return fmt.Errorf("%s is install-wide — use: zenv config set --global %s <value>", key, key)
			}
			return config.Set(key, value)
		},
	}
	setCmd.Flags().BoolVar(&flagGlobal, "global", false, "write install-wide settings (api_url, auth_url) or shared secrets (discouraged)")
	setCmd.Flags().StringVar(&flagProjectID, "project", "", "target project ID for token/project_key")

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			var val string

			switch {
			case config.IsLocalOnly(key) && !flagGlobal:
				val = config.GetLocal(key)
			case config.IsSecret(key):
				if flagGlobal {
					val = config.ListGlobal()[key]
				} else {
					projectID := flagProjectID
					if projectID == "" {
						projectID = config.ResolveProject(flagProject)
					}
					if projectID == "" {
						return fmt.Errorf("no project context — set project in .zenv or use --project")
					}
					val = config.GetForProject(projectID, key)
				}
			default:
				if !flagGlobal {
					return fmt.Errorf("use --global for %s", key)
				}
				val = config.Get(key)
			}

			if val == "" {
				return fmt.Errorf("key %q is not set", key)
			}
			fmt.Println(val)
			return nil
		},
	}
	getCmd.Flags().BoolVar(&flagGlobal, "global", false, "read from global config")
	getCmd.Flags().StringVar(&flagProjectID, "project", "", "read from a specific project's credentials")

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List config for this repo and active project",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			if flagGlobal {
				printKV(w, config.ListGlobal())
				return w.Flush()
			}

			globalKV := loadNonSecretGlobal()
			localKV := config.ListLocal()
			projectID := flagProjectID
			if projectID == "" {
				projectID = config.ResolveProject(flagProject)
			}

			if len(globalKV) > 0 {
				fmt.Fprintln(w, "# global (~/.config/zenv/config)")
				printKV(w, globalKV)
			}
			if len(localKV) > 0 {
				if len(globalKV) > 0 {
					fmt.Fprintln(w)
				}
				fmt.Fprintln(w, "# repo (.zenv)")
				printKV(w, localKV)
			}
			if projectID != "" {
				projKV := config.ListForProject(projectID)
				if len(projKV) > 0 {
					fmt.Fprintln(w)
					fmt.Fprintf(w, "# project %s\n", projectID)
					printKV(w, projKV)
				}
			}

			if len(globalKV) == 0 && len(localKV) == 0 && projectID == "" {
				fmt.Fprintln(os.Stderr, "No config set.")
			}
			return w.Flush()
		},
	}
	listCmd.Flags().BoolVar(&flagGlobal, "global", false, "list only global config")
	listCmd.Flags().StringVar(&flagProjectID, "project", "", "include a specific project's credentials")

	profilesCmd := &cobra.Command{
		Use:     "profiles",
		Short:   "List projects with saved credentials",
		Aliases: []string{"projects"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, err := config.ListConfiguredProjects()
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Println("No project credentials saved.")
				fmt.Println("Run: zenv login")
				return nil
			}

			active := config.ResolveProject(flagProject)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROJECT\tTOKEN\tPROJECT_KEY\tPATH")
			for _, id := range ids {
				kv := config.ListForProject(id)
				token := "—"
				pk := "—"
				if kv[config.KeyToken] != "" {
					token = "yes"
				}
				if kv[config.KeyProjectKey] != "" {
					pk = "yes"
				}
				path, _ := config.ProjectCredentialsPath(id)
				row := id
				if id == active {
					row += " *"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", row, token, pk, path)
			}
			fmt.Fprintln(os.Stderr, "\n* = active project (.zenv or --project)")
			return w.Flush()
		},
	}

	unsetCmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			if config.IsLocalOnly(key) {
				if flagGlobal {
					return fmt.Errorf("%s is per-repo — omit --global", key)
				}
				return config.UnsetLocal(key)
			}

			if config.IsSecret(key) {
				if flagGlobal {
					return config.UnsetGlobalSecret(key)
				}
				projectID := flagProjectID
				if projectID == "" {
					projectID = config.ResolveProject(flagProject)
				}
				if projectID == "" {
					return fmt.Errorf("no project context")
				}
				return config.UnsetForProject(projectID, key)
			}

			if !flagGlobal {
				return fmt.Errorf("use --global for %s", key)
			}
			return config.Unset(key)
		},
	}
	unsetCmd.Flags().BoolVar(&flagGlobal, "global", false, "remove from global config")
	unsetCmd.Flags().StringVar(&flagProjectID, "project", "", "target project ID")

	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Print config directory path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.Dir())
		},
	}

	cmd.AddCommand(setCmd, getCmd, listCmd, unsetCmd, pathCmd, profilesCmd)
	return cmd
}

func loadNonSecretGlobal() map[string]string {
	all := config.ListGlobal()
	out := make(map[string]string)
	for k, v := range all {
		if !config.IsSecret(k) {
			out[k] = v
		}
	}
	return out
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
