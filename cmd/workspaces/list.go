package cmd

import (
	"fmt"

	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
)

// listWorkspacesCmd represents perun deployed workspaces
var listWorkspacesCmd = &cobra.Command{
	Use:   "list",
	Short: "list existing perun workspaces",
	Run: func(cmd *cobra.Command, args []string) {

		utils.Logger = utils.GetLogger(true, "", "")
		workspaces, err := runList()
		cobra.CheckErr(err)
		for _, ws := range workspaces {
			fmt.Printf("Workspace '%s' with %d environments : \n", ws.Name, len(ws.Environments))
			for _, env := range ws.Environments {
				fmt.Printf("	Environment '%s - %s'\n", env.Name, env.Status)
				for _, srv := range env.Services {
					fmt.Printf("		Service '%s - %s'\n", srv.Name, srv.Status)
				}
			}
		}

	},
}

func init() {
	cmd.RootCmd.AddCommand(listWorkspacesCmd)
}
