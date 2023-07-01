package cmd

import (
	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
)

// synchronizeEnvironmentCmd represents perun environment synchronization
var synchronizeEnvironmentCmd = &cobra.Command{
	Use:   "synchronize",
	Short: "synchronize Perun environment in a target workspace",
	Run: func(cmd *cobra.Command, args []string) {
		verbosity, err := cmd.Flags().GetBool("verbose")
		cobra.CheckErr(err)

		workspace, err := cmd.Flags().GetString("workspace")
		cobra.CheckErr(err)

		name, err := cmd.Flags().GetString("env-name")
		cobra.CheckErr(err)

		utils.Logger = utils.GetLogger(verbosity, "Synchronizing environment...", "")
		utils.Logger.Increment(10, "")
		err = runSynchronize(workspace, name)
		utils.Logger.Finish()
		cobra.CheckErr(err)
	},
}

func init() {
	cmd.RootCmd.AddCommand(synchronizeEnvironmentCmd)
}
