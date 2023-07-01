package cmd

import (
	"fmt"
	"github.com/perun-cloud-inc/perunctl/cmd"
	"github.com/perun-cloud-inc/perunctl/utils"
	"github.com/spf13/cobra"
)

// importEnvironmentCmd represents a command to import perun workspaces
var importEnvironmentCmd = &cobra.Command{
	Use:   "import",
	Short: "import target environment into a workspace",
	Run: func(cmd *cobra.Command, args []string) {
		verbosity, err := cmd.Flags().GetBool("verbose")
		cobra.CheckErr(err)

		workspace, err := cmd.Flags().GetString("workspace")
		cobra.CheckErr(err)
		if workspace == "" {
			workspace = "default"
		}

		targetType, err := cmd.Flags().GetString("type")
		cobra.CheckErr(err)
		if targetType == "" {
			targetType = "local"
		}

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

		if targetType == "local" {

			path, err := cmd.Flags().GetString("path")
			cobra.CheckErr(err)
			if path == "" {
				cobra.CheckErr(fmt.Errorf("path arg missing for local import type"))
			}

			name, err := cmd.Flags().GetString("name")
			cobra.CheckErr(err)
			if name == "" {
				cobra.CheckErr(fmt.Errorf("name arg missing for local import type"))
			}
			utils.Logger = utils.GetLogger(verbosity, "Importing local environment...", "")
			utils.Logger.Increment(10, "")
			_, err = workspaceService.ImportLocalEnvironment(workspace, name, path, dbType, dbURL)
			cobra.CheckErr(err)

		} else if targetType == "k8s" {

			name, err := cmd.Flags().GetString("name")
			cobra.CheckErr(err)
			if name == "" {
				cobra.CheckErr(fmt.Errorf("name arg missing for k8s import type"))
			}

			cluster, err := cmd.Flags().GetString("cluster")
			cobra.CheckErr(err)
			if cluster == "" {
				cobra.CheckErr(fmt.Errorf("cluster arg missing for k8s import type"))
			}

			server, err := cmd.Flags().GetString("server")
			cobra.CheckErr(err)

			token, err := cmd.Flags().GetString("token")
			cobra.CheckErr(err)

			ca, err := cmd.Flags().GetString("ca")
			cobra.CheckErr(err)

			excludeList, err := cmd.Flags().GetStringSlice("exclude")
			cobra.CheckErr(err)
			utils.Logger = utils.GetLogger(verbosity, "Importing K8S environment...", "")
			utils.Logger.Increment(10, "")
			_, err = workspaceService.ImportK8sEnvironment(workspace, cluster, name, server, token, ca, excludeList, dbType, dbURL)
			cobra.CheckErr(err)
		}

		utils.Logger.Finish()

	},
}

func init() {
	cmd.RootCmd.AddCommand(importEnvironmentCmd)
	importEnvironmentCmd.Flags().StringP("workspace", "w", "default", "perun target workspace name")
	importEnvironmentCmd.Flags().StringP("type", "t", "local", "target environment type, local or k8s are supported. defaults to local")
	importEnvironmentCmd.Flags().StringP("path", "p", "", "local environment path")
	importEnvironmentCmd.Flags().StringP("name", "n", "", "environment name for local type or k8s namespace for k8s type")
	importEnvironmentCmd.Flags().StringP("cluster", "c", "", "k8s cluster")
	importEnvironmentCmd.Flags().StringP("server", "", "", "k8s server")
	importEnvironmentCmd.Flags().StringP("token", "", "", "k8s token")
	importEnvironmentCmd.Flags().StringP("ca", "", "", "k8s certificate authority")
	importEnvironmentCmd.Flags().StringSliceP("exclude", "e", []string{}, "k8s services to exclude")
	importEnvironmentCmd.Flags().StringP("db-type", "", "", "db type to load (mysql, postgres)")
	importEnvironmentCmd.Flags().StringP("db-url", "", "", "db url in the correct db specific format with the credentials if needed")
	importEnvironmentCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
}
