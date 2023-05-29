package cmd

import (
	// "encoding/json"

	"fmt"
	"os"

	"github.com/spf13/cobra"
	"main.go/model"
	perun_services "main.go/services"
	"main.go/utils"
)

var workspaceService = perun_services.GetWorkspaceService()
var configGeneratorService = getConfigGeneratorService()

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
			_, err := perun_services.GetWorkspaceService().ImportLocalEnvironment(wsName, envName, envPath, dbType, dbURL)
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

	rootCmd.AddCommand(listWorkspacesCmd)

	rootCmd.AddCommand(initWorkspaceCmd)
	initWorkspaceCmd.Flags().StringP("workspace", "w", "", "perun workspace name")
	initWorkspaceCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	initWorkspaceCmd.MarkFlagRequired("workspace")

	rootCmd.AddCommand(applyWorkspaceCmd)
	applyWorkspaceCmd.Flags().StringP("workspace", "w", "", "perun workspace name, if empty try and fetch from provided env and if not set to default")
	applyWorkspaceCmd.Flags().StringP("env-name", "e", "", "environment name, overriding the provided env name in path")
	applyWorkspaceCmd.Flags().StringP("env-path", "p", "", "environment path to load and apply on workspace")
	applyWorkspaceCmd.Flags().BoolP("dry-run", "d", false, "just generate a dry run output in case it was provided...")
	applyWorkspaceCmd.Flags().StringP("db-type", "", "", "db type to load (mysql, postgres)")
	applyWorkspaceCmd.Flags().StringP("db-url", "", "", "db url in the correct db specific format with the credentials if needed")
	applyWorkspaceCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	applyWorkspaceCmd.MarkFlagRequired("env-path")

	rootCmd.AddCommand(destroyWorkspaceCmd)
	destroyWorkspaceCmd.Flags().StringP("workspace", "w", "", "perun workspace name")
	destroyWorkspaceCmd.Flags().StringP("env-name", "e", "", "environment name to destroy, if not provided all environment under workspace will be destroyed")
	destroyWorkspaceCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	destroyWorkspaceCmd.MarkFlagRequired("workspace")

	rootCmd.AddCommand(generateConfigCmd)
	generateConfigCmd.Flags().StringP("workspace", "w", "default", "perun workspace name, if empty set to default")
	generateConfigCmd.Flags().StringP("env-name", "e", "", "target environment name")
	generateConfigCmd.Flags().StringP("service-name", "s", "", "target service name")
	generateConfigCmd.Flags().StringP("ide", "i", "vscode", "target configuration type, defaults to vscode")
	generateConfigCmd.Flags().StringP("source-location", "l", "", "source code path")
	generateConfigCmd.Flags().StringP("source-type", "t", "", "source code programing language (python/node supported)")
	generateConfigCmd.Flags().StringP("command", "c", "", "command to execute to run the application")

	generateConfigCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	generateConfigCmd.MarkFlagRequired("env-name")
	generateConfigCmd.MarkFlagRequired("service-name")

	rootCmd.AddCommand(activateEnvironmentCmd)
	activateEnvironmentCmd.Flags().StringP("workspace", "w", "default", "perun target workspace name")
	activateEnvironmentCmd.Flags().StringP("env-name", "e", "", "perun environment to activate")
	activateEnvironmentCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	activateEnvironmentCmd.MarkFlagRequired("env-name")

	rootCmd.AddCommand(deactivateEnvironmentCmd)
	deactivateEnvironmentCmd.Flags().StringP("workspace", "w", "default", "perun target workspace name")
	deactivateEnvironmentCmd.Flags().StringP("env-name", "e", "", "perun environment to deactivate")
	deactivateEnvironmentCmd.Flags().BoolP("verbose", "v", false, "verbose logger")
	deactivateEnvironmentCmd.MarkFlagRequired("env-name")

	rootCmd.AddCommand(importEnvironmentCmd)
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

	// rootCmd.AddCommand(synchronizeEnvironmentCmd)
	// synchronizeEnvironmentCmd.Flags().String("workspace", "default", "perun target workspace name")
	// synchronizeEnvironmentCmd.Flags().String("name", "", "perun environment to synchronize")
	// synchronizeEnvironmentCmd.MarkFlagRequired("name")

}

func runWorkspaceCreation(name string) (*model.Workspace, error) {
	return workspaceService.CreateWorkspace(name)
}

func runDestroy(name string) error {
	return workspaceService.DestroyWorkspace(name)
}

func runDestroyEnvironment(workspace string, envName string) error {
	return workspaceService.DestroyEnvironment(workspace, envName)
}

func runList() ([]*model.Workspace, error) {
	return workspaceService.ListWorkspaces()
}

func runActivation(workspace string, envName string) error {
	return workspaceService.ActivateEnvironment(workspace, envName)
}

func runDeactivation(workspace string, envName string) error {
	return workspaceService.DeactivateEnvironment(workspace, envName)
}

func runSynchronize(workspace string, envName string) error {
	return workspaceService.SynchronizeEnvironment(workspace, envName)
}

func runGenerateConfig(workspace string, envName string, serviceName string, configType string, sourcecodePath string, sourcecodeType string, command string) (string, error) {
	return configGeneratorService.Generate(workspace, envName, serviceName, configType, sourcecodePath, sourcecodeType, command)
}

func getConfigGeneratorService() perun_services.ConfigGenerator {

	return &perun_services.LocalConfigGenerator{
		WorkspaceService: perun_services.GetWorkspaceService(),
	}
}
