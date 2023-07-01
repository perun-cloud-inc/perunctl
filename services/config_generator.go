package services

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/perun-cloud-inc/perunctl/model"
	"github.com/perun-cloud-inc/perunctl/utils"
)

type ConfigGenerator interface {
	Generate(workspaceName string, environmentName string, serviceName string, configType string, sourcecodePath string, sourcecodeType string, command string) (string, error)
}

type LocalConfigGenerator struct {
	WorkspaceService WorkspacesService
}

func (cg *LocalConfigGenerator) Generate(workspaceName string, environmentName string, serviceName string, configType string, sourcecodePath string, sourcecodeType string, command string) (string, error) {

	ws, err := cg.WorkspaceService.GetWorkspace(workspaceName)

	if err != nil {
		return "", fmt.Errorf("failed to generate config for service %s : %v", serviceName, err)
	}

	if ws == nil {
		return "", fmt.Errorf("failed to generate config for service %s : workspace %s not found", serviceName, workspaceName)
	}

	var environment *model.Environment
	for _, env := range ws.Environments {

		if env.Name == environmentName {

			environment = env

			break
		}

	}

	if environment == nil {

		return "", fmt.Errorf("failed to generate config for service %s : environment %s not found", serviceName, environmentName)
	}

	var service *model.Service
	for _, srv := range environment.Services {

		if srv.Name == serviceName {

			service = srv

			break
		}

	}

	if service == nil {
		return "", fmt.Errorf("failed to generate config for service %s : service not found", serviceName)
	}

	if sourcecodePath != "" {
		service.Params["location"] = sourcecodePath
	}
	if sourcecodeType != "" {
		service.Params["source"] = sourcecodeType
	}

	if command != "" {
		commandArr := strings.Split(command, " ")
		service.Run.Cmd = commandArr[0]
		service.Run.Args = commandArr[1:]
	}

	if configType != "vscode" {
		return "", fmt.Errorf("failed to generate config for service %s , not supported config type %s", serviceName, configType)
	} else {
		generator := VSCodeConfigGenerator{}

		utils.Logger.Info("Generating VSCode debug configuration for %s", serviceName)
		utils.Logger.Increment(10, "")
		return generator.Generate(environment, service)
	}

}

type VSCodeConfigGenerator struct {
}

func (*VSCodeConfigGenerator) GetLaunchConfig(environment *model.Environment, service *model.Service) (*VSCodeLaunchConfig, error) {

	var launchConfig = &VSCodeLaunchConfig{

		Version: "0.2.0",
		Configurations: []VSCodeConfiguration{
			{
				Name:                      fmt.Sprintf("Perun Service %s Debug", service.Name),
				Type:                      "docker",
				Request:                   "launch",
				RemoveContainerAfterDebug: true,
				PreLaunchTask:             "docker-run: debug",
			},
		},
	}

	switch service.Params["source"] {
	case "python":
		launchConfig.Configurations[0].Python = &VSCodeConfigPython{
			ProjectType: "django", //TODO fetch type from code analysis
			PathMappings: []map[string]string{{
				"localRoot":  "${workspaceFolder}",
				"remoteRoot": "/app",
			}},
		}
	case "node":
		launchConfig.Configurations[0].Node = &VSCodeConfigNode{
			RemoteRoot: "/app",
		}
	case "":
		return nil, fmt.Errorf("source type not found for service %s", service.Name)
	default:
		return nil, fmt.Errorf("unsupported source type %s", service.Params["source"])
	}

	return launchConfig, nil

}

func (*VSCodeConfigGenerator) GetTaskConfig(environment *model.Environment, service *model.Service) (*VSCodeTasksConfig, error) {

	var taskConfig = &VSCodeTasksConfig{
		Version: "2.0.0",
		Tasks:   []*VSCodeTask{},
	}

	dockerBuild := &VSCodeTask{
		Type:  "docker-build",
		Label: "docker-build",
		// Platform: "python", //TODO retrieve type from code analysis
		DockerBuild: &VSCodeDockerBuild{
			Tag:           fmt.Sprintf("%s%s%s:latest", environment.Workspace, environment.Name, service.Name),
			DockerFile:    "${workspaceFolder}/Dockerfile", //TODO retrieve from user config and default to this value
			Context:       "${workspaceFolder}",
			Pull:          true,
			CustomOptions: "--platform linux/amd64",
		},
	}

	dockerRun := &VSCodeTask{
		Type:      "docker-run",
		Label:     "docker-run: debug",
		DependsOn: []string{"docker-build"},
		DockerRun: &VSCodeDockerRun{
			Env:   getEnvVarsMap(service.Run.EnVars),
			Image: fmt.Sprintf("%s%s:latest", environment.Workspace+environment.Name, service.Name),
			Labels: map[string]string{
				"perun-workspace":  environment.Workspace,
				"perun-env":        environment.Name,
				"perun-env-target": environment.Target.Type,
				"perun-service":    service.Name,
				"provider":         "perun",
				"provider-mode":    "debug",
			},
			Network:        "default",
			PortPublishAll: true,
			Ports:          getPortMappings(environment, service),
			Volumes: []VSCodeConfigVolumeMapping{
				{
					ContainerPath: "/app",
					LocalPath:     "${workspaceFolder}",
				},
			},
		},
	}

	if service.Params["source"] == "python" {
		dockerBuild.Platform = "python"

		command := service.Run.Cmd
		args := service.Run.Args

		if service.Run.Cmd == "python" {
			command = service.Run.Args[0]
			args = service.Run.Args[1:]
		}

		dockerRun.Python = &VSCodePythonExec{
			File: command,
			Args: args,
		}
	} else if service.Params["source"] == "node" {

		// {
		// 	"type": "docker-run",
		// 	"label": "docker-run: debug",
		// 	"dependsOn": ["docker-build"],
		// 	"dockerRun": {
		// 	  "command": "nest start --debug 0.0.0.0:9229"
		// 	},
		// 	"node": {
		// 	  "enableDebugging": true
		// 	}
		// }
		dockerRun.Node = &VSCodeNodeExec{
			EnableDebugging: true,
		}

		if service.Run.Cmd == "node" {
			dockerRun.DockerRun.Command = fmt.Sprintf("%s --inspect=0.0.0.0:9229 %s", service.Run.Cmd, strings.Join(service.Run.Args, " "))

			// 	debugPortmapping := VSCodeConfigPortMapping{
			// 		ContainerPort: "9229",
			// 	}

			// 	dockerRun.DockerRun.Ports = append(dockerRun.DockerRun.Ports, debugPortmapping)
		} else if service.Run.Cmd == "nest" {
			dockerRun.DockerRun.Command = service.Run.Cmd
			for _, arg := range service.Run.Args {
				if arg == "start" {
					arg += " --debug=0.0.0.0:9229"
				}
				dockerRun.DockerRun.Command += " " + arg
			}
		} else {
			dockerRun.DockerRun.Command = fmt.Sprintf("node  --inspect=0.0.0.0:9229 %s%s", service.Run.Cmd, strings.Join(service.Run.Args, " "))
		}

		dockerRun.DockerRun.CustomOptions = "--entrypoint=\"\" "

	}

	dockerRun.DockerRun.Network = environment.Workspace
	dockerRun.DockerRun.NetworkAlias = service.Name
	dockerRun.DockerRun.CustomOptions += "--workdir=/app"

	dockerRun.DockerRun.Labels["perun-env-target"] = environment.Target.Type

	taskConfig.Tasks = []*VSCodeTask{dockerBuild, dockerRun}

	return taskConfig, nil

}

func (g *VSCodeConfigGenerator) Generate(environment *model.Environment, service *model.Service) (string, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := fmt.Sprintf("%s%s%s/%s/%s/vscode", dirname, utils.WorkspacesHome, environment.Workspace, environment.Name, service.Name)
	utils.Logger.Info("Generating VSCode Launch Config for service %s", service.Name)
	if err := os.MkdirAll(configPath, os.ModePerm); err != nil {
		return "", err
	}

	launchConfig, err := g.GetLaunchConfig(environment, service)

	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(launchConfig, "", "    ")

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create vscode launch config for service %s : %v", service.Name, err)
		return "", err
	}

	err = os.WriteFile(configPath+"/launch.json", data, 0666)
	if err != nil {
		return "", err
	}
	utils.Logger.Increment(20, "")

	if service.Params["location"] != "" {
		if err := os.MkdirAll(service.Params["location"]+"/.vscode/", os.ModePerm); err != nil {
			return "", err
		}
		err = os.WriteFile(service.Params["location"]+"/.vscode/launch.json", data, 0666)
		if err != nil {
			return "", err
		}
	}

	utils.Logger.Info("Generating VSCode Task Config for service %s", service.Name)

	taskConfig, err := g.GetTaskConfig(environment, service)

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create vscode task config for service %s : %v", service.Name, err)
		return "", err
	}

	data, err = json.MarshalIndent(taskConfig, "", "    ")

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create vscode task config for service %s : %v", service.Name, err)
		return "", err
	}

	err = os.WriteFile(configPath+"/tasks.json", data, 0666)

	if err != nil {
		return "", err
	}

	if service.Params["location"] != "" {
		err = os.WriteFile(service.Params["location"]+"/.vscode/tasks.json", data, 0666)
		if err != nil {
			return "", err
		}
	}

	if service.Params["location"] != "" {
		utils.Logger.Info("VSCode launch/tasks configuration generated under %s", service.Params["location"]+"/.vscode")
		return service.Params["location"] + "/.vscode", nil
	}
	utils.Logger.Increment(30, "")

	utils.Logger.Info("VSCode launch/tasks configuration generated under %s", configPath)
	return configPath, nil

}

func getPortMappings(environment *model.Environment, service *model.Service) []VSCodeConfigPortMapping {

	mappings := make([]VSCodeConfigPortMapping, 0)

	for _, portConfig := range service.Run.Ports {

		mapping := VSCodeConfigPortMapping{
			ContainerPort: portConfig.Port,
		}
		if portConfig.Exposed {
			if environment.Target.Type != "docker" {
				mapping.HostPort = portConfig.HostPort
			}
			mappings = append(mappings, mapping)
		}

	}

	return mappings

}

func getEnvVarsMap(envVars []model.EnVar) map[string]string {
	envVarsMap := make(map[string]string, 0)

	for _, envVar := range envVars {
		envVarsMap[envVar.Key] = envVar.Value
	}

	return envVarsMap

}

type VSCodeConfigPython struct {
	PathMappings []map[string]string `json:"pathMappings"`
	ProjectType  string              `json:"projectType"`
}

type VSCodeConfigNode struct {
	RemoteRoot string `json:"remoteRoot"`
}

type VSCodeConfiguration struct {
	Name                      string              `json:"name"`
	Type                      string              `json:"type"`
	Request                   string              `json:"request"`
	Command                   string              `json:"command,omitempty"`
	Program                   string              `json:"program,omitempty"`
	Args                      []string            `json:"args,omitempty"`
	Django                    bool                `json:"django,omitempty"`
	RemoveContainerAfterDebug bool                `json:"removeContainerAfterDebug,omitempty"`
	PreLaunchTask             string              `json:"preLaunchTask,omitempty"`
	PostDebugTask             string              `json:"postDebugTask,omitempty"`
	Platform                  string              `json:"platform,omitempty"`
	Node                      *VSCodeConfigNode   `json:"node,omitempty"`
	Python                    *VSCodeConfigPython `json:"python,omitempty"`
	Env                       map[string]string   `json:"env"`
}

type VSCodeLaunchConfig struct {
	Version        string                `json:"version"`
	Configurations []VSCodeConfiguration `json:"configurations"`
}

type VSCodeDockerBuild struct {
	Tag           string `json:"tag"`
	DockerFile    string `json:"dockerfile"`
	Context       string `json:"context"`
	Pull          bool   `json:"pull"`
	CustomOptions string `json:"customOptions,omitempty"`
}

type VSCodeConfigPortMapping struct {
	ContainerPort string `json:"containerPort"`
	HostPort      string `json:"hostPort"`
}

type VSCodeConfigVolumeMapping struct {
	ContainerPath string `json:"containerPath"`
	LocalPath     string `json:"localPath"`
}

type VSCodeDockerRun struct {
	Env            map[string]string           `json:"env"`
	Image          string                      `json:"image"`
	Labels         map[string]string           `json:"labels"`
	Network        string                      `json:"network"`
	PortPublishAll bool                        `json:"portsPublishAll,omitempty"`
	NetworkAlias   string                      `json:"networkAlias,omitempty"`
	Ports          []VSCodeConfigPortMapping   `json:"ports"`
	Volumes        []VSCodeConfigVolumeMapping `json:"volumes"`
	Command        string                      `json:"command,omitempty"`
	CustomOptions  string                      `json:"customOptions,omitempty"`
}

type VSCodePythonExec struct {
	File string   `json:"file"`
	Args []string `json:"args"`
}

type VSCodeNodeExec struct {
	EnableDebugging bool `json:"enableDebugging"`
}

type VSCodeTask struct {
	Type            string             `json:"type"`
	Label           string             `json:"label"`
	Platform        string             `json:"platform,omitempty"`
	DockerBuild     *VSCodeDockerBuild `json:"dockerBuild,omitempty"`
	DockerRun       *VSCodeDockerRun   `json:"dockerRun,omitempty"`
	DependsOn       []string           `json:"dependsOn,omitempty"`
	Python          *VSCodePythonExec  `json:"python,omitempty"`
	Node            *VSCodeNodeExec    `json:"node,omitempty"`
	Resource        string             `json:"resource,omitempty"`
	ResourceType    string             `json:"resourceType,omitempty"`
	TargetCluster   string             `json:"targetCluster,omitempty"`
	TargetNamespace string             `json:"targetNamespace,omitempty"`
	Ports           []int              `json:"ports,omitempty"`
	Cmd             string             `json:"command,omitempty"`
	Args            []string           `json:"args,omitempty"`
	Env             map[string]string  `json:"env,omitempty"`
}

// type BridgeVSCodeTask struct {
// 	VSCodeTask
// 	Resource        string `json:"resource,omitempty"`
// 	ResourceType    string `json:"resourceType,omitempty"`
// 	TargetCluster   string `json:"targetCluster,omitempty"`
// 	TargetNamespace string `json:"targetNamespace,omitempty"`
// 	Ports           []int  `json:"ports"`
// }

type VSCodeTasksConfig struct {
	Version string        `json:"version"`
	Tasks   []*VSCodeTask `json:"tasks"`
}
