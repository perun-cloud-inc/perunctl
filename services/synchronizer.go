package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	ps "github.com/mitchellh/go-ps"
	"main.go/model"
	"main.go/utils"
)

type SynchronizationService interface {
	Synchronize(*model.Environment) error
	Unsynchronize(*model.Environment) error
	Destroy(*model.Environment) error
}

type DockerSynchronizationService struct {
}

//service type -
//service source : git/local/docker
//service build config: how to build the executable and the wrraping docker
//service run config : how to run the app .... cmmnd and args

func (s DockerSynchronizationService) Synchronize(env *model.Environment) error {

	allocation := utils.Logger.GetProgressAllocation(env.Name)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: env.Workspace}),
	})

	var targetNetworkID string

	if len(networks) > 1 {
		return fmt.Errorf("failed to synchronize env %s, Too many networks with the same workspace name %s", env.Name, env.Workspace)
	} else if len(networks) == 1 {
		targetNetworkID = networks[0].ID

	} else if len(networks) == 0 {
		resp, err := cli.NetworkCreate(ctx, env.Workspace, types.NetworkCreate{
			CheckDuplicate: true,
			Attachable:     true,
		})
		if err != nil {
			return err
		}

		targetNetworkID = resp.ID
	}

	// load db container first
	dbService := env.Services["perun-db"]
	if dbService != nil {
		loadService(ctx, cli, targetNetworkID, env, dbService)
		time.Sleep(15 * time.Second)
		if dbService.Build.Type == "db" {
			dbType := dbService.Build.Params["type"]
			dbURL := dbService.Build.Params["url"]
			targetDBURL := dbService.Build.Params["target-url"]

			location, err := getDumpLocation(env, dbService)
			if err != nil {
				return err
			}
			var dumper DatabaseCopy
			if dbType == "mysql" {

				dumper = MySQLCopy{
					URL:         dbURL,
					TargetFile:  location,
					TargetDBURL: targetDBURL,
				}

			} else if dbType == "postgres" {

				dumper = PostgresCopy{
					URL:         dbURL,
					TargetFile:  location,
					TargetDBURL: targetDBURL,
				}

			} else {
				utils.Logger.Warn("cannot load db, unsupported db type %s", dbType)
			}

			if dumper != nil {
				err := dumper.Copy()
				if err != nil {
					return err
				}
			}
		}
	}

	increment := 0
	if len(env.Services) > 0 {
		increment = allocation / len(env.Services)
	}
	for _, service := range env.Services {

		if service.Name == "perun-db" {
			continue
		}

		loadService(ctx, cli, targetNetworkID, env, service)
		utils.Logger.Increment(increment, "")

	}

	// go ContainerEvents(cli)
	env.Status = model.ACTIVE_STATUS
	return nil

}

func loadService(ctx context.Context, cli *client.Client, targetNetworkID string, env *model.Environment, service *model.Service) error {

	runConfig := service.Run
	containerID := ""
	imageName := ""

	if service.Build != nil && service.Build.Type == "dockerfile" {
		//TODO build custom container first
		//docker file support?
	}

	volumeLocalPath := ""
	switch service.Type {
	case "local":
		imageName = service.Params["image"] //in case an image was supplied for a local setup , we load that image. location will be used for debug capability
		if imageName == "" {
			switch service.Params["source"] {
			case "python":
				imageName = "python"
			case "node":
				imageName = "node"
			case "":
				//TODO analyze local code - should move that into import flow .. import will fetch git, analyze and set attributes for user to validate
				_, err := analyzeCodeSource(service.Params["location"])
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported source type %s", service.Params["source"])
			}

			version := service.Params["version"]
			if version != "" {
				imageName += ":" + version
			}
			volumeLocalPath = service.Params["location"]
		}

	case "docker":

		imageName = service.Params["image"]

	case "git":
		//TODO code fetch ?
		//TODO code source analysis

		return fmt.Errorf("not supported service type %s", service.Type)

	default:
		return fmt.Errorf("not supported service type %s", service.Type)
	}

	pullOptions := types.ImagePullOptions{
		Platform: "linux/amd64",
	}

	authConfig := service.ContainerRegistry
	if authConfig == nil {
		authConfig = env.ContainerRegistry
	}
	if authConfig != nil {
		encodedJSON, err := json.Marshal(types.AuthConfig{
			Username:      authConfig.Username,
			Password:      authConfig.Password,
			RegistryToken: authConfig.Token,
		})
		if err != nil {
			return err
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		pullOptions.RegistryAuth = authStr

	}

	reader, err := cli.ImagePull(ctx, imageName, pullOptions)
	if err != nil {
		return err
	}

	io.Copy(utils.Logger.GetOutput(), reader)

	config := &container.Config{

		Image: imageName,
		Env:   GetEnVars(service.Run.EnVars),
		// WorkingDir: "/app",
		Labels: map[string]string{"provider": "perun", "provider-mode": "sync", "perun-workspace": env.Workspace, "perun-env": env.Name, "perun-env-target": "docker", "perun-service": service.Name},
	}

	hostConfig := &container.HostConfig{
		Runtime:    "runc",
		AutoRemove: false,
		RestartPolicy: container.RestartPolicy{
			MaximumRetryCount: 10,
		},
	}

	exposedPortsArr := []string{}
	portsArr := []string{}
	if len(service.Run.Ports) > 0 {
		for _, port := range service.Run.Ports {
			if port.Exposed {
				exposedPortsArr = append(exposedPortsArr, port.HostPort+":"+port.Port)
			} else {
				portsArr = append(portsArr, port.Port)
			}

		}

		exposedPorts, bindings, _ := nat.ParsePortSpecs(exposedPortsArr)

		ports, _, _ := nat.ParsePortSpecs(portsArr)

		for k, v := range exposedPorts {
			ports[k] = v
		}

		if len(ports) > 0 {
			config.ExposedPorts = ports
		}

		if len(bindings) > 0 {
			hostConfig.PortBindings = bindings
		}

	}

	if runConfig.Cmd != "" {

		cmmnd := []string{"/bin/sh", "-c"}

		generaterdCmmnd := ""
		if len(service.PreRun) > 0 {

			for _, pc := range service.PreRun {

				if imageName == "python" && strings.HasSuffix(pc.Cmd, ".py") {
					generaterdCmmnd += "python "
				}
				generaterdCmmnd += pc.Cmd
				if len(pc.Args) > 0 {
					generaterdCmmnd += " " + strings.Join(pc.Args, " ")
				}
				generaterdCmmnd += " && "

			}

		}

		if imageName == "python" && strings.HasSuffix(runConfig.Cmd, ".py") {
			generaterdCmmnd += "python "
		}

		generaterdCmmnd += runConfig.Cmd

		if len(runConfig.Args) > 0 {
			generaterdCmmnd += " " + strings.Join(runConfig.Args, " ")
		}

		if strings.HasPrefix(runConfig.Cmd, "/bin/sh") { //TODO minimize whitespace for correct split
			cmmnd = strings.SplitN(generaterdCmmnd, " ", 3)
		} else {
			cmmnd = append(cmmnd, generaterdCmmnd)
		}

		config.Cmd = cmmnd
	}

	hostConfig.Mounts = []mount.Mount{}
	if volumeLocalPath != "" {
		hostConfig.Mounts = []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: volumeLocalPath,
				Target: "/app",
			},
		}
	}

	if len(service.Run.Mounts) > 0 {
		for _, serviceMount := range service.Run.Mounts {
			prpFilesLocation, err := getPropertiesFilesLocation(env, service, serviceMount)
			if err != nil {
				return err
			}
			mountElem := mount.Mount{
				Type:   mount.TypeBind,
				Source: prpFilesLocation,
				Target: serviceMount.Path,
			}

			hostConfig.Mounts = append(hostConfig.Mounts, mountElem)
		}

	}

	plt := &v1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, plt, env.Name+"-"+service.Name)
	if err != nil {
		return err
	}
	containerID = resp.ID

	cli.NetworkConnect(ctx, targetNetworkID, containerID, &network.EndpointSettings{
		Aliases: []string{service.Name},
	})

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	service.Status = model.ACTIVE_STATUS

	return nil

}

func (s DockerSynchronizationService) Listen() error {

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	ContainerEvents(cli)

	return nil
}

func getDumpLocation(env *model.Environment, service *model.Service) (string, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create temp properties folder, failed to fetch home directory : %v", err)
		return "", err
	}
	workspacesDirectory := dirname + utils.WORKSPACES_HOME + env.Workspace

	dumpsFileLocation := workspacesDirectory + "/" + service.Name + "/dump/"
	err = os.MkdirAll(dumpsFileLocation, os.ModePerm)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create dump folder: %v", err)
		return "", err
	}

	return dumpsFileLocation + service.Name, nil
}

func getPropertiesFilesLocation(env *model.Environment, service *model.Service, mount model.Mount) (string, error) {

	configsFileLocation := mount.SourcePath
	var err error
	if configsFileLocation == "" {

		dirname, err := os.UserHomeDir()
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed to create temp properties folder, failed to fetch home directory : %v", err)
			return "", err
		}
		workspacesDirectory := dirname + utils.WORKSPACES_HOME + env.Workspace

		configsFileLocation = workspacesDirectory + "/" + service.Name + "/properties/" + mount.Name + "/"
		err = os.MkdirAll(configsFileLocation, os.ModePerm)
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed to create temp properties folder, failed to create properties folder : %v", err)
			return "", err
		}

	} else {
		if strings.HasSuffix(configsFileLocation, "/") == false {
			configsFileLocation += "/"
		}
	}

	configs := make([]string, 0)
	for _, config := range mount.Configs {

		configLocation := configsFileLocation + config.ConfigName
		configs = append(configs, configLocation)
		data := []byte(config.Content)
		err = os.WriteFile(configLocation, data, os.ModePerm)
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed to create service properties file %s : %v", configLocation, err)
			return "", err
		}
	}

	return configsFileLocation, nil
}

func isDebugConnect(msg events.Message) bool {
	return msg.Type == "container" && msg.Action == "start" && msg.Actor.Attributes["provider"] == "perun" && msg.Actor.Attributes["provider-mode"] == "debug"
}

func isDebugDisconnect(msg events.Message) bool {
	return msg.Type == "container" && msg.Action == "destroy" && msg.Actor.Attributes["provider"] == "perun" && msg.Actor.Attributes["provider-mode"] == "debug"
}

func ContainerEvents(client *client.Client) error {

	utils.Logger.Info("Starting Docker Event listener")

	msgChannel, errs := client.Events(context.Background(), types.EventsOptions{
		Filters: filters.NewArgs(),
	})

	for {
		select {
		case err := <-errs:
			utils.Logger.Error("%v", err)
			// TODO handle error
			return err
		case msg := <-msgChannel:
			// log.Info(fmt.Sprintf("Type : %s\n", msg.Type))
			// log.Info(fmt.Sprintf("Action : %s\n", msg.Action))
			// log.Info(fmt.Sprintf("Attributes : %v\n\n\n", msg.Actor.Attributes))
			if isDebugConnect(msg) {
				if msg.Actor.Attributes["perun-env-target"] == "docker" || msg.Actor.Attributes["perun-env-target"] == "local" {

					originalContainerName := msg.Actor.Attributes["perun-env"] + "-" + msg.Actor.Attributes["perun-service"]

					inspect, err := client.ContainerInspect(context.TODO(), originalContainerName)
					if err != nil {
						utils.Logger.Error("%v", err)
						return err
					}

					if inspect.State.Running {
						// stop running containerz

						// err := client.NetworkDisconnect(context.TODO(), msg.Actor.Attributes["perun-workspace"], originalContainerName, true)
						// if err != nil {
						// 	utils.Logger.Error("%v", err)
						// 	return err
						// }

						// err = client.NetworkConnect(context.TODO(), msg.Actor.Attributes["perun-workspace"], originalContainerName, &network.EndpointSettings{
						// 	Aliases: []string{},
						// })
						// if err != nil {
						// 	utils.Logger.Error("%v", err)
						// 	return err
						// }

						err = client.ContainerStop(context.TODO(), originalContainerName, nil)

						if err != nil {
							utils.Logger.Error("%v", err)
							return err
						}
					} else {
						utils.Logger.Warn("Failed to find a running %s container", originalContainerName)
					}
				}

			} else if isDebugDisconnect(msg) {

				if msg.Actor.Attributes["perun-env-target"] == "docker" || msg.Actor.Attributes["perun-env-target"] == "local" {
					// start stopped container
					originalContainerName := msg.Actor.Attributes["perun-env"] + "-" + msg.Actor.Attributes["perun-service"]

					inspect, err := client.ContainerInspect(context.TODO(), originalContainerName)
					if err != nil {
						utils.Logger.Error("%v", err)
						return err
					}

					if inspect.State.Status == "exited" {

						// err := client.NetworkDisconnect(context.TODO(), msg.Actor.Attributes["perun-workspace"], originalContainerName, true)
						// if err != nil {
						// 	utils.Logger.Error("%v", err)
						// 	return err
						// }

						// err = client.NetworkConnect(context.TODO(), msg.Actor.Attributes["perun-workspace"], originalContainerName, &network.EndpointSettings{
						// 	Aliases: []string{msg.Actor.Attributes["perun-service"]},
						// })
						// if err != nil {
						// 	utils.Logger.Error("%v", err)
						// 	return err
						// }

						err = client.ContainerStart(context.TODO(), originalContainerName, types.ContainerStartOptions{})
						if err != nil {
							utils.Logger.Error("%v", err)
							return err
						}
					} else {
						utils.Logger.Warn("Failed to find a paused %s container", originalContainerName)
					}
				}

			}
		}
	}

}

func KillDSCProcess() (bool, error) {

	processList, err := ps.Processes()
	if err != nil {
		return false, fmt.Errorf("failed to retrieve running processes to kill linguering dsc")
	}

	// map ages
	for x := range processList {
		var process = processList[x]
		utils.Logger.Info("%d\t%s\n", process.Pid(), process.Executable())
		if process.Executable() == "dsc" {
			proc, err := os.FindProcess(process.Pid())
			if err != nil {
				return false, fmt.Errorf("failed to kill lingering dsc process")
			}
			// Kill the process
			err = proc.Kill()
			return true, err
		}
	}

	return false, nil
}

func (s DockerSynchronizationService) Unsynchronize(env *model.Environment) error {

	err := s.Destroy(env)
	if err != nil {
		return fmt.Errorf("Failed to deactivate environment %s/%s", env.Workspace, env.Name)
	}

	env.Status = model.INACTIVE_STATUS
	return nil
}

func (s DockerSynchronizationService) Destroy(env *model.Environment) error {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	allocation := utils.Logger.GetProgressAllocation(env.Name)
	increment := 0
	if len(env.Services) > 0 {
		increment = allocation / len(env.Services)
	}

	for _, service := range env.Services {

		utils.Logger.Info("destroying service %s/%s/%s", env.Workspace, env.Name, service.Name)
		containers, err := cli.ContainerList(ctx, types.ContainerListOptions{

			Filters: filters.NewArgs(filters.Arg("name", env.Name+"-"+service.Name)),
		})

		if err != nil {
			return err
		}

		containersCount := len(containers)
		var containerToDelete types.Container
		if containersCount != 1 {

			if containersCount == 0 {

				// log as warning .. nothing to destroy, no container found
				continue

			} else {
				duplicates := 0
				for _, container := range containers {
					if container.Names[0] == "/"+env.Name+"-"+service.Name {
						duplicates++
						containerToDelete = container
					}
				}
				if duplicates > 1 {
					return fmt.Errorf("too many contianers with name %s", env.Name+"-"+service.Name)
				}
			}
		} else {
			containerToDelete = containers[0]
		}

		containerID := containerToDelete.ID

		if err := cli.ContainerStop(ctx, containerID, nil); err != nil {
			return err
		}

		utils.Logger.Info("stopping container %s with ID %s", env.Name+"-"+service.Name, containerID)

		if err := cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false,
			Force:         true,
		}); err != nil {
			return err
		}

		utils.Logger.Info("removing container %s with ID %s", env.Name+"-"+service.Name, containerID)

		service.Status = model.INACTIVE_STATUS

		utils.Logger.Increment(increment, "")

	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", env.Workspace)),
	})

	networksCount := len(networks)
	if networksCount != 1 {

		return fmt.Errorf("too many networks with name %s", env.Workspace)

	}

	targetNetworkID := networks[0].ID

	cli.NetworkRemove(ctx, targetNetworkID)
	return nil

}

func GetEnVars(envars []model.EnVar) []string {
	envarArr := make([]string, 0)
	for _, envar := range envars {
		envarArr = append(envarArr, envar.Key+"="+envar.Value)
	}

	return envarArr
}

func analyzeCodeSource(location string) (string, error) {

	return "", fmt.Errorf("code analysis not yet supported")

}
