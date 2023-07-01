package cmd

import (
	"fmt"

	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/services"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
)

// applyCmd represents perun empty workspace creation
var applyWorkspaceCmd = &cobra.Command{
	Use:   "apply",
	Short: "apply the provided env on a workspace, in dry run mode the environment will be analyzed and persisted but not loaded into the target deployment",
	Run: func(cmd *cobra.Command, args []string) {
		verbosity, err := cmd.Flags().GetBool("verbose")
		cobra.CheckErr(err)
		wsName, err := cmd.Flags().GetString("workspace")
		cobra.CheckErr(err)

		envName, err := cmd.Flags().GetString("env-name")
		cobra.CheckErr(err)

		envPath, err := cmd.Flags().GetString("env-path")
		cobra.CheckErr(err)

		dbType, err := cmd.Flags().GetString("db-type")
		cobra.CheckErr(err)

		dbURL, err := cmd.Flags().GetString("db-url")
		cobra.CheckErr(err)
		if dbURL != "" {
			if dbType == "" {
				dbType = "mysql"
			}
		}

		if dbURL == "" && dbType != "" {
			cobra.CheckErr(fmt.Errorf("db-type arg was provided without any db-url"))
		}

		utils.Logger = utils.GetLogger(verbosity, "Applying workspace...", "")
		utils.Logger.Increment(10, "")
		if wsName != "" {
			ws, err := workspaceService.GetWorkspace(wsName)
			cobra.CheckErr(err)
			if ws == nil {
				_, err = runWorkspaceCreation(wsName)
				cobra.CheckErr(err)
			}
		}

		if envPath != "" {
			_, err := services.GetWorkspaceService().ImportLocalEnvironment(wsName, envName, envPath, dbType, dbURL)
			cobra.CheckErr(err)

			dryRun, err := cmd.Flags().GetBool("dry-run")
			cobra.CheckErr(err)
			if !dryRun {
				err = runActivation(wsName, envName)
				cobra.CheckErr(err)
			}

		}

		utils.Logger.Finish()
	},
}

func init() {
	cmd.RootCmd.AddCommand(applyWorkspaceCmd)
	applyWorkspaceCmd.Flags().StringP("workspace", "w", "", "perun workspace name, if empty try and fetch from provided env and if not set to default")
	applyWorkspaceCmd.Flags().StringP("env-name", "e", "", "environment name, overriding the provided env name in path")
	applyWorkspaceCmd.Flags().StringP("env-path", "p", "", "environment path to load and apply on workspace")
	applyWorkspaceCmd.Flags().BoolP("dry-run", "d", false, "just generate a dry run output in case it was provided...")
	applyWorkspaceCmd.Flags().StringP("db-type", "", "", "db type to load (mysql, postgres)")
	applyWorkspaceCmd.Flags().StringP("db-url", "", "", "db url in the correct db specific format with the credentials if needed")
	applyWorkspaceCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	err := applyWorkspaceCmd.MarkFlagRequired("env-path")
	if err != nil {
		// TODO: Handle it
		return
	}
}
