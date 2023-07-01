package cmd

import (
	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
)

// deactivateEnvironmentCmd represents perun environment deactivation
var deactivateEnvironmentCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "deactivate Perun environment in a target workspace",
	Run: func(cmd *cobra.Command, args []string) {
		verbosity, err := cmd.Flags().GetBool("verbose")
		cobra.CheckErr(err)

		wsName, err := cmd.Flags().GetString("workspace")
		cobra.CheckErr(err)

		envName, err := cmd.Flags().GetString("env-name")
		cobra.CheckErr(err)
		utils.Logger = utils.GetLogger(verbosity, "Deactivating environment...", "")
		utils.Logger.Increment(10, "")
		err = runDeactivation(wsName, envName)
		utils.Logger.Finish()
		cobra.CheckErr(err)
	},
}

func init() {
	cmd.RootCmd.AddCommand(deactivateEnvironmentCmd)
	deactivateEnvironmentCmd.Flags().StringP("workspace", "w", "default", "perun target workspace name")
	deactivateEnvironmentCmd.Flags().StringP("env-name", "e", "", "perun environment to deactivate")
	deactivateEnvironmentCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	err := deactivateEnvironmentCmd.MarkFlagRequired("env-name")
	if err != nil {
		// TODO: Handle it
		return
	}
}
