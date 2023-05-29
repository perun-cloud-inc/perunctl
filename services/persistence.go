package services

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"

	"main.go/model"
	"main.go/utils"
)

// An interface representing a persistence abstraction whether for a workspace or an environment
type PersistenceService interface {
	WorkspacePersistenceService
}

// An interface representing a persistence abstraction whether for a workspace
type WorkspacePersistenceService interface {
	PersistWorkspace(*model.Workspace) error
	ClearWorkspace(*model.Workspace) error
	GetWorkspace(name string) (*model.Workspace, error)
	ListWorkspaces() ([]*model.Workspace, error)
	LoadEnvironment(envPath string) (*model.Environment, error)
}

type LocalPersistenceService struct {
}

func (ps LocalPersistenceService) LoadEnvironment(envPath string) (*model.Environment, error) {
	// log.Info("Loading environment from path %s", envPath)
	exists, err := Exists(envPath)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to fetch environment %s, no environment yaml found : %v", envPath, err)
		return nil, err

	}
	if !exists {
		err = fmt.Errorf("failed to fetch environment %s, no environment yaml found found", envPath)
		return nil, err

	} else {

		yfile, err := os.ReadFile(envPath)

		if err != nil {
			err = fmt.Errorf("failed to fetch environment at %s, error reading environment yaml", envPath)
			return nil, err
		}

		data := model.Environment{}

		err = yaml.Unmarshal(yfile, &data)

		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed to fetch environment at %s, error deserializing environment yaml", envPath)
			return nil, err
		}

		return &data, nil

	}

}

func (ps LocalPersistenceService) PersistWorkspace(ws *model.Workspace) error {

	//Build workspace folder structure
	dirname, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create workspace %s work dirs : %v", ws.Name, err)
		return err
	}

	wsLocation := dirname + utils.WORKSPACES_HOME + ws.Name
	if err := os.MkdirAll(wsLocation, os.ModePerm); err != nil {

		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create workspace %s work dirs : %v", ws.Name, err)
		return err

	}

	// file, err := os.OpenFile(ws_location + "/workspace.yml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	// if err != nil {
	//     log.Fatalf("error opening/creating file: %v", err)
	// }
	// defer file.Close()

	data, err := yaml.Marshal(ws)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create workspace %s work dirs, failed to serialize workspace : %v", ws.Name, err)
		return err
	}

	err = os.WriteFile(wsLocation+"/workspace.yml", data, 0666)

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to create workspace %s work dirs, failed to persist to file : %v", ws.Name, err)
		return err
	}

	// fmt.Printf("--- t dump:\n%s\n\n", string(d))
	return nil
}

func (ps LocalPersistenceService) ClearWorkspace(ws *model.Workspace) error {

	//Clear workspace folder structure
	dirname, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to clear workspace %s work dirs : %v", ws.Name, err)
		return err
	}

	utils.Logger.Info("Clearing workspace %s, at %s", ws.Name, dirname+utils.WORKSPACES_HOME+ws.Name)
	if err = os.RemoveAll(dirname + utils.WORKSPACES_HOME + ws.Name); err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to clear workspace %s work dirs : %v", ws.Name, err)
		return err
	}

	return nil
}

func (ps LocalPersistenceService) GetWorkspace(name string) (*model.Workspace, error) {

	dirname, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to list workspaces, failed to fetch home directory : %v", err)
		return nil, err
	}

	workspaceLocation := dirname + utils.WORKSPACES_HOME + name + "/workspace.yml"
	exists, err := Exists(workspaceLocation)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to fetch workspace %s, no workspace yaml found : %v", workspaceLocation, err)
		return nil, err

	}
	if !exists {

		return nil, nil

	}

	yfile, err := os.ReadFile(workspaceLocation)

	if err != nil {
		err = fmt.Errorf("failed to fetch workspaces %s, error reading workspace yaml", workspaceLocation)
		return nil, err
	}

	data := model.Workspace{}

	err = yaml.Unmarshal(yfile, &data)

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to fetch workspaces %s, error deserializing workspace yaml", workspaceLocation)
		return nil, err
	}

	return &data, nil

}

func (ps LocalPersistenceService) ListWorkspaces() ([]*model.Workspace, error) {

	workspaces := make([]*model.Workspace, 0)
	dirname, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to list workspaces, failed to fetch home directory : %v", err)
		return nil, err
	}
	workspacesDirectory := dirname + utils.WORKSPACES_HOME

	if err := os.MkdirAll(workspacesDirectory, os.ModePerm); err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to init workspaces directory : %v", err)
		return nil, err
	}
	files, err := ioutil.ReadDir(workspacesDirectory)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to list workspaces, failed to read workspaces directory %s : %v", workspacesDirectory, err)
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			workspaceLocation := workspacesDirectory + file.Name() + "/workspace.yml"
			exists, err := Exists(workspaceLocation)
			if err != nil {
				utils.Logger.Warn("failed to fetch workspaces %s, no workspace yaml found : %v", workspacesDirectory+file.Name(), err)
				continue
			}
			if !exists {
				utils.Logger.Warn("failed to fetch workspaces %s, no workspace yaml found", workspacesDirectory+file.Name())
				continue
			} else {

				yfile, err := os.ReadFile(workspaceLocation)

				if err != nil {
					utils.Logger.Warn("failed to fetch workspaces %s, error reading workspace yaml", workspaceLocation)
					continue
				}

				data := model.Workspace{}

				err = yaml.Unmarshal(yfile, &data)

				if err != nil {
					utils.Logger.Warn("failed to fetch workspaces %s, error deserializing workspace yaml", workspaceLocation)
					continue
				}

				workspaces = append(workspaces, &data)

			}

		}

	}

	return workspaces, nil
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
