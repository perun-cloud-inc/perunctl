package cmd

import (
	"fmt"
	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
	"os"
)

// generateConfigCmd represents a command to generate perun debug config
var generateConfigCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate debug config for supplied service",
	Run: func(cmd *cobra.Command, args []string) {
		verbosity, err := cmd.Flags().GetBool("verbose")
		cobra.CheckErr(err)

		workspace, err := cmd.Flags().GetString("workspace")
		cobra.CheckErr(err)
		if workspace == "" {
			workspace = "default"
		}

		environment, err := cmd.Flags().GetString("env-name")
		cobra.CheckErr(err)

		service, err := cmd.Flags().GetString("service-name")
		cobra.CheckErr(err)

		configType, err := cmd.Flags().GetString("ide")
		cobra.CheckErr(err)

		sourcecodePath, err := cmd.Flags().GetString("source-location")
		cobra.CheckErr(err)

		if stat, err := os.Stat(sourcecodePath); os.IsNotExist(err) && !stat.IsDir() {
			cobra.CheckErr(fmt.Errorf("provided source code path %s should point to an existing folder path", sourcecodePath))
		}

		sourcecodeType, err := cmd.Flags().GetString("source-type")
		cobra.CheckErr(err)
		if sourcecodeType != "" && sourcecodeType != "python" && sourcecodeType != "node" {
			cobra.CheckErr(fmt.Errorf("unsupported source code language %s.. currently only python and nodejs are supported", sourcecodeType))
		}
		command, err := cmd.Flags().GetString("command")
		cobra.CheckErr(err)

		utils.Logger = utils.GetLogger(verbosity, "Generating debug configuration...", "")
		utils.Logger.Increment(10, "")
		_, err = runGenerateConfig(workspace, environment, service, configType, sourcecodePath, sourcecodeType, command)
		utils.Logger.Finish()
		cobra.CheckErr(err)
	},
}

func init() {
	cmd.RootCmd.AddCommand(generateConfigCmd)
	generateConfigCmd.Flags().StringP("workspace", "w", "default", "perun workspace name, if empty set to default")
	generateConfigCmd.Flags().StringP("env-name", "e", "", "target environment name")
	generateConfigCmd.Flags().StringP("service-name", "s", "", "target service name")
	generateConfigCmd.Flags().StringP("ide", "i", "vscode", "target configuration type, defaults to vscode")
	generateConfigCmd.Flags().StringP("source-location", "l", "", "source code path")
	generateConfigCmd.Flags().StringP("source-type", "t", "", "source code programing language (python/node supported)")
	generateConfigCmd.Flags().StringP("command", "c", "", "command to execute to run the application")

	generateConfigCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	err := generateConfigCmd.MarkFlagRequired("env-name")
	if err != nil {
		// TODO: Handle it
		return
	}
	err = generateConfigCmd.MarkFlagRequired("service-name")
	if err != nil {
		// TODO: Handle it
		return
	}
}
