package cmd

import (
	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
)

// activateEnvironmentCmd represents perun environment activation
var activateEnvironmentCmd = &cobra.Command{
	Use:   "activate",
	Short: "activate Perun environment in a target workspace",
	Run: func(cmd *cobra.Command, args []string) {
		verbosity, err := cmd.Flags().GetBool("verbose")
		cobra.CheckErr(err)

		wsName, err := cmd.Flags().GetString("workspace")
		cobra.CheckErr(err)

		envName, err := cmd.Flags().GetString("env-name")
		cobra.CheckErr(err)

		utils.Logger = utils.GetLogger(verbosity, "Activating environment...", "")
		utils.Logger.Increment(10, "")
		err = runActivation(wsName, envName)
		utils.Logger.Finish()
		cobra.CheckErr(err)
	},
}

func init() {
	cmd.RootCmd.AddCommand(activateEnvironmentCmd)
	activateEnvironmentCmd.Flags().StringP("workspace", "w", "default", "perun target workspace name")
	activateEnvironmentCmd.Flags().StringP("env-name", "e", "", "perun environment to activate")
	activateEnvironmentCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	err := activateEnvironmentCmd.MarkFlagRequired("env-name")
	if err != nil {
		// TODO: handle it
		return
	}
}
