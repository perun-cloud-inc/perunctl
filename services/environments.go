package services

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	ps "github.com/mitchellh/go-ps"
	"github.com/perun-cloud-inc/perunctl/model"
	"github.com/perun-cloud-inc/perunctl/utils"
)

type EnvironmentService interface {
	CreateEnvironment(name string) (*model.Environment, error)
	ActivateEnvironment(env *model.Environment) error
	DeactivateEnvironment(env *model.Environment) error
	DestroyEnvironment(env *model.Environment) error
	SyncEnvironment(env *model.Environment) error
}

type LocalEnvironmentService struct {
	ValidationService      EnvironmentValidationService
	SynchronizationService SynchronizationService
}

func (es LocalEnvironmentService) CreateEnvironment(name string) (*model.Environment, error) {

	utils.Logger.Info("Creating environment %s", name)
	var env = model.Environment{}
	return &env, nil
}

func LoadEventsListener() error {

	up, err := CheckEventsListener()

	if err != nil {
		return err
	}

	if up {
		return nil
	}

	binaryLocation, err := CheckEventsListenerLocation()
	if err != nil {
		utils.Logger.Fatal("%v", err)
	}

	cmd := exec.Command(binaryLocation)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		utils.Logger.Fatal("cmd.Start failed: %v", err)
	}
	err = cmd.Process.Release()
	if err != nil {
		utils.Logger.Fatal("cmd.Process.Release failed: %v", err)
	}

	return nil

}

func CheckEventsListenerLocation() (string, error) {

	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	exPath := filepath.Dir(ex)
	binPath := fmt.Sprintf("%s/events", exPath)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	if _, err := os.Stat(binPath); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("Perun events binary doesn't exist under %s", binPath)
	}

	return binPath, nil

}

func CheckEventsListener() (bool, error) {

	processList, err := ps.Processes()
	if err != nil {
		return false, fmt.Errorf("failed to retrieve running processes to initiate event listener")
	}

	// map ages
	for x := range processList {
		var process = processList[x]
		utils.Logger.Info("%d\t%s\n", process.Pid(), process.Executable())
		if process.Executable() == "events" {
			return true, nil
		}
	}

	return false, nil
}

func (es LocalEnvironmentService) ActivateEnvironment(env *model.Environment) error {

	utils.Logger.Info("Activating environment %s", env.Name)

	go func() {
		err := LoadEventsListener()
		if err != nil {
			utils.Logger.Error("failed to load perun events listener")
		}

	}()

	if env.Status == model.ActiveStatus && env.Target.Type == "docker" {
		return fmt.Errorf("target environment %s is already in active state", env.Name)
	}

	err := es.ValidationService.ValidateEnvironment(env)
	if err != nil {
		return err
	}

	for _, service := range env.Services {

		err = es.ValidationService.ValidateService(service)
		if err != nil {
			return err
		}

	}

	err = es.SynchronizationService.Synchronize(env)
	if err != nil {
		return err
	}

	return nil
}

func (es LocalEnvironmentService) DeactivateEnvironment(env *model.Environment) error {

	utils.Logger.Info("Deactivating environment %s", env.Name)
	if env.Status != model.ActiveStatus {
		return fmt.Errorf("deactivation aborted, Target environment %s not in active state", env.Name)
	}

	err := es.SynchronizationService.Unsynchronize(env)
	if err != nil {
		return err
	}

	return nil
}

func (es LocalEnvironmentService) DestroyEnvironment(env *model.Environment) error {

	utils.Logger.Info("Destroying environment %s", env.Name)

	if env.Status != model.ActiveStatus {
		if env.Status == model.InactiveStatus {
			utils.Logger.Info("Environment %s is in inactive state... skipping environment deletion", env.Name)
			return nil
		}
		return fmt.Errorf("aborted environment deletion, Target environment %s not in active state", env.Name)
	}

	if env.Target.Type != "local" {
		return nil
	}
	err := es.SynchronizationService.Destroy(env)
	if err != nil {
		return err
	}

	return nil

}

func (es LocalEnvironmentService) SyncEnvironment(env *model.Environment) error {

	utils.Logger.Info("Synching environment %s", env.Name)
	err := es.ValidationService.ValidateEnvironment(env)
	if err != nil {
		return err
	}

	for _, service := range env.Services {

		err = es.ValidationService.ValidateService(service)
		if err != nil {
			return err
		}

	}

	err = es.SynchronizationService.Synchronize(env)
	if err != nil {
		return err
	}

	return nil
}
