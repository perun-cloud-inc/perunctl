package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"main.go/model"
	"main.go/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/strings/slices"
)

type EnvironmentsMap map[string]*model.Environment

var Workspaces = make(map[string]*model.Workspace)
var Environments = make(map[string]EnvironmentsMap)

// service for workspaces management
type WorkspacesService interface {
	CreateWorkspace(name string) (*model.Workspace, error)
	GetWorkspace(name string) (*model.Workspace, error)
	ListWorkspaces() ([]*model.Workspace, error)
	DestroyWorkspace(name string) error
	ActivateEnvironment(workspace string, environment string) error
	DeactivateEnvironment(workspace string, environment string) error
	SynchronizeEnvironment(workspace string, environment string) error
	DestroyEnvironment(workspace string, environment string) error
}

type LocalWorkspacesService struct {
	// workspaces         map[string]*model.Workspace
	ValidationService  WorkspaceValidationService
	PersistenceService WorkspacePersistenceService
	EnvironmentService EnvironmentService
	AnalyzerService    AnalyzerService
}

func (wss LocalWorkspacesService) CreateWorkspace(name string) (*model.Workspace, error) {
	utils.Logger.Info("Creating workspace %s", name)
	var ws = &model.Workspace{
		Name: name,
		Mode: model.Local.String(),
	}

	err := wss.ValidationService.ValidateWorkspace(ws)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed to validate workspace %s for create operation, : %v", name, err)

		return nil, err

	}
	utils.Logger.Increment(20, "")
	err = wss.PersistenceService.PersistWorkspace(ws)

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed persisting workspace %s : %v", name, err)

		return nil, err

	}

	Workspaces[ws.Name] = ws
	utils.Logger.Increment(70, "")
	utils.Logger.Info("workspace %s created successfully", name)
	return ws, nil
}

func (wss LocalWorkspacesService) GetWorkspace(name string) (*model.Workspace, error) {

	utils.Logger.Info("fetching workspace %s", name)

	ws, err := wss.PersistenceService.GetWorkspace(name)
	if err != nil {
		err = fmt.Errorf("failed to fetch workspace %s, workspace not found", name)
		return nil, err
	}

	return ws, nil
}

func (wss LocalWorkspacesService) ListWorkspaces() ([]*model.Workspace, error) {

	utils.Logger.Info("Listing workspaces")
	return wss.PersistenceService.ListWorkspaces()
}

func (wss LocalWorkspacesService) DestroyWorkspace(name string) error {

	utils.Logger.Info("Destroying workspace %s", name)

	ws, err := wss.GetWorkspace(name)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed destroying workspace %s : %v", name, err)

		return err

	}

	if ws == nil {
		err = fmt.Errorf("failed destroying workspace %s, workspace not found", name)
		return err
	}

	increment := 10
	if len(ws.Environments) > 0 {
		increment = (utils.Logger.ProgressLeft - 10) / len(ws.Environments)
	}
	for _, env := range ws.Environments {

		utils.Logger.Info("destroying environment %s under workspace %s", env.Name, ws.Name)

		utils.Logger.SetProgressAllocation(env.Name, increment)
		err = wss.EnvironmentService.DestroyEnvironment(env)
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed destroying workspace %s, failed to destroy  environment %s  %v", name, env.Name, err)
			return err

		}
		utils.Logger.Info("environment %s under workspace %s was destroyed", env.Name, ws.Name)
		utils.Logger.Increment(increment, "")

	}

	err = wss.PersistenceService.ClearWorkspace(ws)

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed destroying workspace %s : %v", name, err)
		return err

	}
	utils.Logger.Increment(10, "")
	delete(Workspaces, ws.Name)

	utils.Logger.Info("Workspace %s destroyed successfully", name)
	return nil
}

func CheckConnection(server string, token string, ca string) (*kubernetes.Clientset, error) {

	var config *rest.Config
	var err error

	if server == "" && token == "" && ca == "" {
		// creates the in-cluster config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("couldn't load default k8s config, unable to retrive user home dir: %w", err)
		}
		config, err = clientcmd.BuildConfigFromFlags("", homeDir+"/.kube/config")
		if err != nil {
			return nil, fmt.Errorf("couldn't load default k8s config: %w", err)
		}

		// configLoadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: homeDir + "/.kube/config"}
		// configOverrides := &clientcmd.ConfigOverrides{CurrentContext: "test-cluster-1"}
		// config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, configOverrides).ClientConfig()
		// if err != nil {
		// 	return nil, err
		// }

		// config, err = rest.InClusterConfig()
		// if err != nil {
		// 	return nil, fmt.Errorf("couldn't load default k8s config: %w", err)
		// }

	} else {

		cert, err := base64.StdEncoding.DecodeString(ca)
		if err != nil {
			return nil, fmt.Errorf("invalid certificate cluster=%s cert=%s: %w", server, ca, err)
		}

		config = &rest.Config{
			Host:        server,
			BearerToken: token,
			TLSClientConfig: rest.TLSClientConfig{
				CAData: cert,
			},
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	_, err = clientset.ServerVersion()

	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func getPodsForSvc(svc *corev1.Service, namespace string, k8sClient *kubernetes.Clientset) (*corev1.PodList, error) {
	set := labels.Set(svc.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := k8sClient.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	// for _, pod := range pods.Items {
	// 	fmt.Fprintf(os.Stdout, "pod name: %v\n", pod.Name)
	// }
	return pods, err
}

func (wss LocalWorkspacesService) ImportK8sEnvironment(targetWorkspace string, k8sCluster string, k8sNamespace string, k8sServer string, k8sToken string, k8sCertAuth string, excludeList []string, dbType string, dbURL string) (*model.Environment, error) {

	utils.Logger.Info("Importing k8s namespace %s into workspace %s", k8sNamespace, targetWorkspace)
	ws, err := wss.GetWorkspace(targetWorkspace)

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed importing k8s namespace %s, Failed to get target workspace %s : %w", k8sNamespace, targetWorkspace, err)

		return nil, err
	}
	if ws == nil {
		utils.Logger.Info("workspace %s doesn't exist, creating it now", targetWorkspace)
		utils.Logger.IgnoreIncrements(true)
		ws, err = wss.CreateWorkspace(targetWorkspace)
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed creating workspace %s : %v", targetWorkspace, err)
			return nil, err
		}
		utils.Logger.IgnoreIncrements(false)
	}
	utils.Logger.Increment(10, "")

	for _, environment := range ws.Environments {
		if environment.Name == k8sNamespace {
			err = fmt.Errorf("failed importing k8s namespace %s, environment %s already exist in workspace %s", k8sNamespace, k8sNamespace, targetWorkspace)

			return nil, err
		}

	}
	var importedEnv *model.Environment
	clientset, err := CheckConnection(k8sServer, k8sToken, k8sCertAuth)
	if err != nil {
		return nil, err
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(k8sNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	configMapsMap := make(map[string]map[string]string)
	for _, c := range configMaps.Items {

		configMapsMap[c.Name] = c.Data

	}

	k8sServices, err := clientset.CoreV1().Services(k8sNamespace).List(context.TODO(), metav1.ListOptions{})
	if statusError, isStatus := err.(*errors.StatusError); isStatus {
		return nil, fmt.Errorf("error getting services in namespace %s: %v", k8sNamespace, statusError.ErrStatus.Message)
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve namespace %s services : %v", k8sNamespace, err)
	}

	importedEnv = &model.Environment{
		Name:      k8sNamespace,
		Workspace: targetWorkspace,
		Status:    model.INACTIVE_STATUS,
		Target: model.Target{
			Name:   k8sNamespace,
			Type:   "local",
			Params: map[string]string{"namespace": k8sNamespace, "cluster": k8sCluster},
		},
	}

	increment := (utils.Logger.ProgressLeft - 10) / len(k8sServices.Items)
	services := make(map[string]*model.Service)
	for _, k8sService := range k8sServices.Items {

		if slices.Contains(excludeList, k8sService.Name) {
			utils.Logger.Info("Skip importing k8s service %s from namespace %s into workspace %s, service in include list", k8sService.Name, k8sNamespace, targetWorkspace)
			continue
		}
		utils.Logger.Info("Importing k8s service %s from namespace %s into workspace %s", k8sService.Name, k8sNamespace, targetWorkspace)
		service := &model.Service{
			Name:   k8sService.Name,
			Type:   "docker",
			Status: model.INACTIVE_STATUS,
			Params: make(map[string]string),
		}
		pods, err := getPodsForSvc(&k8sService, k8sNamespace, clientset)

		if err != nil {
			utils.Logger.Info("Failed finding relevant pod for k8s service %s from namespace %s : %v", k8sService.Name, k8sNamespace, err)
			continue
		}
		if len(pods.Items) == 0 {
			utils.Logger.Info("Failed finding relevant pod for k8s service %s from namespace %s", k8sService.Name, k8sNamespace)
			continue
		}
		pod := pods.Items[0]

		configVolumes := make(map[string]*corev1.ConfigMapVolumeSource)
		volumes := pod.Spec.Volumes

		for _, v := range volumes {
			if v.ConfigMap != nil {
				configVolumes[v.Name] = v.ConfigMap
			}

		}

		container := pod.Spec.Containers[0]

		service.Run = &model.RunConfig{
			Cmd:    strings.Join(container.Command, " "),
			Args:   container.Args,
			EnVars: make([]model.EnVar, 0),
			Ports:  []model.Port{},
			Mounts: make(map[string]model.Mount),
		}

		for _, v := range container.VolumeMounts {

			if configVolumes[v.Name] != nil {

				mount := model.Mount{
					Name:    v.Name,
					Path:    v.MountPath,
					Configs: make([]model.Config, 0),
				}

				cm := configMapsMap[configVolumes[v.Name].Name]
				for key, value := range cm {

					config := model.Config{
						ConfigName: key,
						Content:    value,
					}
					mount.Configs = append(mount.Configs, config)

				}
				service.Run.Mounts[v.Name] = mount
			}
		}
		service.Params["image"] = container.Image

		for _, k8sServicePort := range k8sService.Spec.Ports {

			portStr := fmt.Sprintf("%d", k8sServicePort.TargetPort.IntVal)

			if err != nil {
				return nil, fmt.Errorf("failed importing k8s namespace %s, failed to convert service %s port", k8sNamespace, service.Name)
			}
			service.Run.Ports = append(service.Run.Ports, model.Port{
				Port:    portStr,
				Exposed: true,
			})

		}

		if service.Run.EnVars == nil {
			service.Run.EnVars = make([]model.EnVar, 0)
		}

		envFromArr := pod.Spec.Containers[0].EnvFrom
		if len(envFromArr) > 0 {
			for _, v := range envFromArr {
				if v.ConfigMapRef != nil {

					cm := configMapsMap[v.ConfigMapRef.Name]
					for key, value := range cm {
						envVar := model.EnVar{
							Key:   key,
							Value: value,
						}
						service.Run.EnVars = append(service.Run.EnVars, envVar)
					}

				}
			}
		}
		envVars := pod.Spec.Containers[0].Env
		for _, v := range envVars {

			key := ""
			value := ""

			if v.ValueFrom != nil {
				if v.ValueFrom.ConfigMapKeyRef != nil {
					key = v.ValueFrom.ConfigMapKeyRef.Key
					value = configMapsMap[v.ValueFrom.ConfigMapKeyRef.Name][v.ValueFrom.ConfigMapKeyRef.Key]
				} else if v.ValueFrom.SecretKeyRef != nil {
					//TODO handle config secret
					secretName := v.ValueFrom.SecretKeyRef.LocalObjectReference.Name
					secretKey := v.ValueFrom.SecretKeyRef.Key
					res, err := clientset.CoreV1().Secrets(k8sNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
					if err != nil {
						return nil, fmt.Errorf("failed importing k8s namespace %s, failed to convert service %s secrets : %v", k8sNamespace, service.Name, err)
					}
					utils.Logger.Debug("secret name %s, secret key %s, value %s", secretName, secretKey, string(res.Data[secretKey]))
					key = v.Name
					value = string(res.Data[secretKey])
				} else {
					utils.Logger.Info("unsupported environment variable %s for service %s", v, service.Name)
					continue
				}

			} else {
				key = v.Name
				value = v.Value
			}

			envVar := model.EnVar{
				Key:   key,
				Value: value,
			}
			service.Run.EnVars = append(service.Run.EnVars, envVar)

		}

		services[service.Name] = service

		if err != nil {
			return nil, fmt.Errorf("error getting pods in namespace %s for service %s: %v", k8sNamespace, k8sService.Name, err)
		}

		utils.Logger.Increment(increment, "")

	}

	importedEnv.Services = services

	// ApplyStatus(importedEnv, model.ACTIVE_STATUS)
	importedEnv.Workspace = ws.Name

	if dbURL != "" && dbType != "" {
		dbService, err := GetDBService(dbURL, dbType, "latest")
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed to retrieve perun db service of type %s, for env %s/%s: %v", dbType, targetWorkspace, k8sNamespace, err)
			return nil, err
		}

		importedEnv.Services["perun-db"] = dbService
	}

	ws.Environments = append(ws.Environments, importedEnv)

	utils.Logger.Info("Persisting imported k8s namespace %s into workspace %s", k8sNamespace, targetWorkspace)

	err = wss.PersistenceService.PersistWorkspace(ws)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed importing k8s namesapce %s into target workspace %s, persistence failed : %v", k8sNamespace, targetWorkspace, err)

		return nil, err
	}

	utils.Logger.Info("Successfully imported k8s namespace %s into workspace %s", k8sNamespace, targetWorkspace)

	return importedEnv, nil

}

func (wss LocalWorkspacesService) ImportLocalEnvironment(targetWorkspace string, envName string, envPath string, dbType string, dbURL string) (*model.Environment, error) {

	utils.Logger.Info("Importing environment in path %s into workspace %s", envPath, targetWorkspace)
	ws, err := wss.GetWorkspace(targetWorkspace)

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed importing environment in path %s, Failed to get target workspace %s : %v", envPath, targetWorkspace, err)

		return nil, err
	}

	if ws == nil {
		ws, err = wss.CreateWorkspace(targetWorkspace)
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed importing environment in path %s, Failed to create target workspace %s : %v", envPath, targetWorkspace, err)

			return nil, err
		}
	}

	exists, err := Exists(envPath)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed importing environment in path %s, Failed to check for existence : %v", envPath, err)

		return nil, err
	}

	if !exists {

		err = fmt.Errorf("failed importing environment in path %s, bad environment location", envPath)

		return nil, err
	}

	utils.Logger.Increment(10, "")

	importedEnv, err := wss.PersistenceService.LoadEnvironment(envPath)
	if envName != "" {
		importedEnv.Name = envName
	}

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed importing environment in path %s, Failed to laod environment : %v", envPath, err)

		return nil, err
	}

	//TODO analyze loaded env and services
	// analyzedEnv, err := wss.AnalyzerService.AnalyzeEnvironment(importedEnv)

	ApplyStatus(importedEnv, model.INACTIVE_STATUS)
	importedEnv.Workspace = ws.Name
	utils.Logger.Increment(10, "")
	for _, environment := range ws.Environments {
		if environment.Name == importedEnv.Name {
			err = fmt.Errorf("failed importing environment in path %s, environment %s already exist in workspace %s", envPath, importedEnv.Name, targetWorkspace)

			return nil, err
		}

	}

	ws.Environments = append(ws.Environments, importedEnv)

	if dbURL != "" && dbType != "" {
		dbService, err := GetDBService(dbURL, dbType, "latest")
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed to retrieve perun db service of type %s, for env %s/%s: %v", dbType, targetWorkspace, envName, err)
			return nil, err
		}

		importedEnv.Services["perun-db"] = dbService
	}

	err = wss.PersistenceService.PersistWorkspace(ws)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed importing environment in path %s into target workspace %s, persistence failed : %v", envPath, targetWorkspace, err)

		return nil, err
	}

	utils.Logger.Increment(10, "")
	// Workspaces[ws.Name] = ws
	// envMap := Environments[ws.Name]

	// envMap[ImportedEnv.Name] = ImportedEnv
	utils.Logger.Info("Environment %s, imported successfully into workspace %s, run activation flow in order to activate it", importedEnv.Name, ws.Name)
	return importedEnv, nil

}

func (wss LocalWorkspacesService) ActivateEnvironment(targetWorkspace string, environment string) error {
	utils.Logger.Info("Activating environment %s/%s", targetWorkspace, environment)
	ws, err := wss.GetWorkspace(targetWorkspace)
	if err != nil {
		return err
	}
	if ws == nil {
		return fmt.Errorf("failed to activate %s/%s, workspace %s not found", targetWorkspace, environment, targetWorkspace)
	}
	var targetEnv *model.Environment
	for _, env := range ws.Environments {
		if env.Name == environment {
			targetEnv = env
			break
		}

	}

	if targetEnv == nil {
		return fmt.Errorf("failed to find target environment %s/%s", targetWorkspace, environment)
	}

	targetEnv.Workspace = ws.Name
	utils.Logger.Increment(10, "")
	utils.Logger.SetProgressAllocation(targetEnv.Name, 60)
	err = wss.EnvironmentService.ActivateEnvironment(targetEnv)
	if err != nil {
		return err
	}

	err = wss.PersistenceService.PersistWorkspace(ws)

	if err != nil {
		return err
	}

	utils.Logger.Increment(10, "")
	utils.Logger.Info("Environment %s/%s activated successfully", targetWorkspace, environment)
	return nil
}

func (wss LocalWorkspacesService) DeactivateEnvironment(targetWorkspace string, environment string) error {

	utils.Logger.Info("Deactivating environment %s/%s", targetWorkspace, environment)
	ws, err := wss.GetWorkspace(targetWorkspace)
	if err != nil {
		return err
	}
	var targetEnv *model.Environment
	for _, env := range ws.Environments {
		if env.Name == environment {
			targetEnv = env
			break
		}

	}

	if targetEnv == nil {
		return fmt.Errorf("failed to find target environment %s under %s workspace", environment, targetWorkspace)
	}
	utils.Logger.Increment(10, "")
	utils.Logger.SetProgressAllocation(targetEnv.Name, 60)
	err = wss.EnvironmentService.DeactivateEnvironment(targetEnv)
	if err != nil {
		return err
	}

	err = wss.PersistenceService.PersistWorkspace(ws)
	if err != nil {
		return err
	}
	utils.Logger.Increment(10, "")
	utils.Logger.Info("Environment %s/%s deactivated successfully", targetWorkspace, environment)
	return nil
}

func (wss LocalWorkspacesService) SynchronizeEnvironment(targetWorkspace string, environment string) error {
	utils.Logger.Info("Synchronizing environment %s/%s", targetWorkspace, environment)
	ws, err := wss.GetWorkspace(targetWorkspace)
	if err != nil {
		return err
	}
	var targetEnv *model.Environment
	for _, env := range ws.Environments {
		if env.Name == environment {
			targetEnv = env
			break
		}
	}

	if targetEnv == nil {
		return fmt.Errorf("failed to find target environment %s under %s workspace", environment, targetWorkspace)
	}

	utils.Logger.SetProgressAllocation(targetEnv.Name, 80)
	err = wss.EnvironmentService.SyncEnvironment(targetEnv)

	if err != nil {
		return err
	}
	utils.Logger.Info("Environment %s/%s synchronized successfully", targetWorkspace, environment)
	return nil
}

func (wss LocalWorkspacesService) DestroyEnvironment(targetWorkspace string, environment string) error {
	utils.Logger.Info("Destroying environment %s/%s", targetWorkspace, environment)

	ws, err := wss.GetWorkspace(targetWorkspace)
	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed destroying environment %s - workspace %s not found : %v", environment, targetWorkspace, err)
		return err
	}

	if ws == nil {
		err = fmt.Errorf("failed destroying workspace %s, workspace not found", targetWorkspace)
		return err
	}
	utils.Logger.SetProgressAllocation(environment, 70)
	remainingEnvs := make([]*model.Environment, 0)
	for _, env := range ws.Environments {

		if env.Name != environment {
			remainingEnvs = append(remainingEnvs, env)
			continue
		}

		err = wss.EnvironmentService.DestroyEnvironment(env)
		if err != nil {
			utils.Logger.Error("%v", err)
			err = fmt.Errorf("failed destroying environment %s under workspace %s : %v", env.Name, targetWorkspace, err)
			return err

		}

		utils.Logger.Info("Environment %s/%s destroyed successfully", targetWorkspace, environment)

	}

	ws.Environments = remainingEnvs

	err = wss.PersistenceService.PersistWorkspace(ws)

	if err != nil {
		utils.Logger.Error("%v", err)
		err = fmt.Errorf("failed persisting workspace %s after env %s was destroyed, need a manual intervention : %v", targetWorkspace, environment, err)

		return err

	}

	return nil
}

func ApplyStatus(env *model.Environment, status string) error {

	env.Status = status
	for _, srv := range env.Services {
		srv.Status = status
	}

	return nil

}
