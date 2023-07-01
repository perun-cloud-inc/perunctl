package cmd

import (
	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
)

// destroyWorkspaceCmd represents perun workspace deletion
var destroyWorkspaceCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroys and clears given workspace",
	Run: func(cmd *cobra.Command, args []string) {
		verbosity, err := cmd.Flags().GetBool("verbose")
		cobra.CheckErr(err)

		wsName, err := cmd.Flags().GetString("workspace")
		cobra.CheckErr(err)

		envName, err := cmd.Flags().GetString("env-name")
		cobra.CheckErr(err)

		if envName == "" {
			utils.Logger = utils.GetLogger(verbosity, "Destroying workspace...", "")
			utils.Logger.Increment(10, "")
			err = runDestroy(wsName)
			cobra.CheckErr(err)
		} else {
			utils.Logger = utils.GetLogger(verbosity, "Destroying environment...", "")
			utils.Logger.Increment(10, "")
			err = runDestroyEnvironment(wsName, envName)
			cobra.CheckErr(err)
		}
		utils.Logger.Finish()
	},
}

func init() {
	cmd.RootCmd.AddCommand(destroyWorkspaceCmd)
	destroyWorkspaceCmd.Flags().StringP("workspace", "w", "", "perun workspace name")
	destroyWorkspaceCmd.Flags().StringP("env-name", "e", "", "environment name to destroy, if not provided all environment under workspace will be destroyed")
	destroyWorkspaceCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	err := destroyWorkspaceCmd.MarkFlagRequired("workspace")
	if err != nil {
		// TODO: Handle it
		return
	}
}
