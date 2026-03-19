package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Judeadeniji/zenv-sh/cli/internal/client"
)

var (
	orgName      string
	memberUserID string
	memberRole   string
)

func newOrgsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "orgs",
		Aliases: []string{"o"},
		Short:   "Manage organizations",
	}

	cmd.AddCommand(
		newOrgsCreateCmd(),
		newOrgsListCmd(),
		newOrgsGetCmd(),
		newOrgsMembersCmd(),
		newOrgsAddMemberCmd(),
		newOrgsRemoveMemberCmd(),
	)

	return cmd
}

func newOrgsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			if orgName == "" {
				return fmt.Errorf("--name is required")
			}
			if api == nil {
				return fmt.Errorf("ZENV_TOKEN is not set.\nSet it: export ZENV_TOKEN=svc_...")
			}

			o, err := api.CreateOrg(orgName)
			if err != nil {
				return err
			}

			fmt.Println("Organization created successfully.")
			fmt.Println("")
			fmt.Printf("  ID:    %s\n", o.ID)
			fmt.Printf("  Name:  %s\n", o.Name)
			fmt.Printf("  Owner: %s\n", o.OwnerID)
			return nil
		},
	}
	cmd.Flags().StringVar(&orgName, "name", "", "organization name (required)")
	return cmd
}

func newOrgsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List your organizations",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				return fmt.Errorf("ZENV_TOKEN is not set.\nSet it: export ZENV_TOKEN=svc_...")
			}

			orgs, err := api.ListOrgs()
			if err != nil {
				return err
			}

			if len(orgs) == 0 {
				fmt.Println("No organizations found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tCREATED")
			for _, o := range orgs {
				fmt.Fprintf(w, "%s\t%s\t%s\n", o.ID, o.Name, o.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
}

func newOrgsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get ORG_ID",
		Short: "Get organization details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				return fmt.Errorf("ZENV_TOKEN is not set.\nSet it: export ZENV_TOKEN=svc_...")
			}

			o, err := api.GetOrg(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("  ID:      %s\n", o.ID)
			fmt.Printf("  Name:    %s\n", o.Name)
			fmt.Printf("  Owner:   %s\n", o.OwnerID)
			fmt.Printf("  Created: %s\n", o.CreatedAt)
			return nil
		},
	}
}

func newOrgsMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "members ORG_ID",
		Short:   "List members of an organization",
		Aliases: []string{"ls-members"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				return fmt.Errorf("ZENV_TOKEN is not set.\nSet it: export ZENV_TOKEN=svc_...")
			}

			members, err := api.ListMembers(args[0])
			if err != nil {
				return err
			}

			if len(members) == 0 {
				fmt.Println("No members found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tEMAIL\tROLE\tJOINED")
			for _, m := range members {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					m.ID[:8]+"...", m.Email, m.Role, m.JoinedAt)
			}
			w.Flush()
			return nil
		},
	}
}

func newOrgsAddMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-member ORG_ID",
		Short: "Add a user to an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if memberUserID == "" {
				return fmt.Errorf("--user is required")
			}
			if memberRole == "" {
				memberRole = "dev"
			}
			if api == nil {
				return fmt.Errorf("ZENV_TOKEN is not set.\nSet it: export ZENV_TOKEN=svc_...")
			}

			m, err := api.AddMember(args[0], client.AddMemberRequest{
				UserID: memberUserID,
				Role:   memberRole,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Added member %s with role %s\n", m.UserID, m.Role)
			return nil
		},
	}
	cmd.Flags().StringVar(&memberUserID, "user", "", "user ID to add (required)")
	cmd.Flags().StringVar(&memberRole, "role", "dev", "role: admin, senior_dev, dev, contractor, ci_bot")
	return cmd
}

func newOrgsRemoveMemberCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-member ORG_ID MEMBER_ID",
		Short: "Remove a member from an organization",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if api == nil {
				return fmt.Errorf("ZENV_TOKEN is not set.\nSet it: export ZENV_TOKEN=svc_...")
			}

			if err := api.RemoveMember(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Member %s removed.\n", args[1])
			return nil
		},
	}
}
