package cmd

import (
	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
)

// initCmd represents perun empty workspace creation
var initWorkspaceCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize empty Perun workspace",
	Run: func(cmd *cobra.Command, args []string) {
		verbosity, err := cmd.Flags().GetBool("verbose")
		cobra.CheckErr(err)
		name, err := cmd.Flags().GetString("workspace")
		cobra.CheckErr(err)
		utils.Logger = utils.GetLogger(verbosity, "Creating workspace...", "")
		utils.Logger.Increment(10, "")
		_, err = runWorkspaceCreation(name)
		utils.Logger.Finish()
		cobra.CheckErr(err)

	},
}

func init() {
	cmd.RootCmd.AddCommand(initWorkspaceCmd)
	initWorkspaceCmd.Flags().StringP("workspace", "w", "", "perun workspace name")
	initWorkspaceCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	err := initWorkspaceCmd.MarkFlagRequired("workspace")
	if err != nil {
		// TODO: Handle it
		return
	}
}
