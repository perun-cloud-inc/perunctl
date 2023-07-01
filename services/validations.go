package services

import (
	"github.com/perun-cloud-inc/perunctl/model"
	"github.com/perun-cloud-inc/perunctl/utils"
)

type ValidationService interface {
	WorkspaceValidationService
	EnvironmentValidationService
}

type WorkspaceValidationService interface {
	ValidateWorkspace(*model.Workspace) error
}

type EnvironmentValidationService interface {
	ServiceValidationService
	ValidateEnvironment(*model.Environment) error
}

type ServiceValidationService interface {
	ValidateService(*model.Service) error
}

type ValidationServiceImpl struct {
}

func (v ValidationServiceImpl) ValidateWorkspace(ws *model.Workspace) error {
	utils.Logger.Info("Validating workspace %s", ws.Name)

	validations, err := v.GetWorkspaceValidations(ws)

	if err != nil {
		//TODO wrap in a descriptive err
		return err
	}

	for _, v := range validations {
		err = v.Validate(ws)
		if err != nil {
			//TODO wrap in a descriptive err
			utils.Logger.Error("Failed validating workspace %s : %v", ws.Name, err)
			return err

		}
	}

	return nil
}

func (v ValidationServiceImpl) ValidateEnvironment(env *model.Environment) error {
	utils.Logger.Info("Validating environment %s", env.Name)
	validations, err := v.GetEnvironmentValidations(env)

	if err != nil {
		//TODO wrap in a descriptive err
		return err
	}

	for _, v := range validations {
		err = v.Validate(env)
		if err != nil {
			//TODO wrap in a descriptive err
			return err

		}
	}

	return nil
}

func (v ValidationServiceImpl) ValidateService(srv *model.Service) error {
	utils.Logger.Info("Validating service %s", srv.Name)
	validations, err := v.GetServiceValidations(srv)

	if err != nil {
		//TODO wrap in a descriptive err
		return err
	}

	for _, v := range validations {
		err = v.Validate(srv)
		if err != nil {
			//TODO wrap in a descriptive err
			return err

		}
	}

	return nil
}

func (ValidationServiceImpl) GetWorkspaceValidations(ws *model.Workspace) ([]WorkspaceValidation, error) {

	return []WorkspaceValidation{}, nil
}

func (ValidationServiceImpl) GetEnvironmentValidations(ws *model.Environment) ([]EnvironmentValidation, error) {
	return []EnvironmentValidation{}, nil
}

func (ValidationServiceImpl) GetServiceValidations(ws *model.Service) ([]ServiceValidation, error) {
	return []ServiceValidation{}, nil
}

type WorkspaceValidation interface {
	Validate(ws *model.Workspace) error
}

type EnvironmentValidation interface {
	Validate(env *model.Environment) error
}

type ServiceValidation interface {
	Validate(service *model.Service) error
}
