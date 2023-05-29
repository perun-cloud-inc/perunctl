package services

import (
	"fmt"

	"main.go/model"
)

type AnalyzerService interface {
	AnalyzeEnvironment(env *model.Environment) (*model.Environment, error)
	AnalyzeService(service *model.Service) (*model.Service, error)
}

type AnalyzerServiceImpl struct {
}

func copyEnv(env *model.Environment) (*model.Environment, error) {

	aenv := &model.Environment{
		Name:        env.Name,
		Description: env.Description,
		Workspace:   env.Workspace,
		Target:      env.Target,
		Services:    make(map[string]*model.Service),
		Status:      env.Status,
	}

	return aenv, nil

}

func copyService(svc *model.Service) (*model.Service, error) {

	asvc := &model.Service{

		Name:        svc.Name,
		Description: svc.Description,
		Type:        svc.Type,
		Status:      svc.Status,
		Params:      svc.Params,
		DependsOn:   svc.DependsOn,
		Build:       svc.Build,
		PreRun:      svc.PreRun,
		Run:         svc.Run,
		PostRun:     svc.PostRun,
	}

	return asvc, nil

}

func (a AnalyzerServiceImpl) AnalyzeEnvironment(env *model.Environment) (*model.Environment, error) {
	aenv, err := copyEnv(env)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze environment %s, side copy env failed", env.Name)
	}
	for _, svc := range env.Services {

		asvc, err := a.AnalyzeService(svc)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze environment %s, analysis of service %s failed", env.Name, svc.Name)
		}
		aenv.Services[asvc.Name] = asvc

	}

	return aenv, nil

}

func (a AnalyzerServiceImpl) AnalyzeService(service *model.Service) (*model.Service, error) {

	if service.Type != "local" {
		return service, nil
	}

	if service.Params["source"] != "" {
		return service, nil
	}

	asvc, err := copyService(service)

	if err != nil {
		return nil, fmt.Errorf("failed to analyze service %s, side copy env failed", service.Name)
	}

	repoLocation := asvc.Params["source"]

	if repoLocation == "" {
		return nil, fmt.Errorf("failed to analyze service %s, no location specified for local service type ", service.Name)
	}

	//TODO analyze service and return source type "python/java/golang..etc" and version if possible
	//TODO for python try to identify framework used flask/django

	return asvc, nil
}
