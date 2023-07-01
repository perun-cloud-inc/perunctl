package cmd

import (
	"github.com/perun-cloud-inc/perunctl/model"
	perunServices "github.com/perun-cloud-inc/perunctl/services"
)

var (
	workspaceService       = perunServices.GetWorkspaceService()
	configGeneratorService = getConfigGeneratorService()
)

func init() {
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

func getConfigGeneratorService() perunServices.ConfigGenerator {

	return &perunServices.LocalConfigGenerator{
		WorkspaceService: perunServices.GetWorkspaceService(),
	}
}
