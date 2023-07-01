package services

import (
	"fmt"
	"strings"

	"net/url"

	"github.com/perun-cloud-inc/perunctl/model"
)

func GetWorkspaceService() LocalWorkspacesService {
	return LocalWorkspacesService{
		ValidationService:  ValidationServiceImpl{},
		PersistenceService: LocalPersistenceService{},
		EnvironmentService: LocalEnvironmentService{
			ValidationService:      ValidationServiceImpl{},
			SynchronizationService: DockerSynchronizationService{},
		},
		AnalyzerService: AnalyzerServiceImpl{},
	}
}

func GetDBService(dbURL string, dbType string, dbVersion string) (*model.Service, error) {

	image, err := GetDBImage(dbType, "latest")
	if err != nil {
		err = fmt.Errorf("failed to create a db service of type %s:%s, failed to retrieve image : %v", dbType, dbVersion, err)
		return nil, err
	}

	// dbEnVars, err := GetDBEnVars(dbURL, dbType, dbVersion)
	// if err != nil {
	// 	err = fmt.Errorf("failed to create a db service of type %s:%s, failed to retrieve env vars : %v", dbType, dbVersion, err)
	// 	return nil, err
	// }

	dbEnVars := make([]model.EnVar, 0)

	_, dbName, dbUser, dbPass, err := GetSQLEnVars(dbURL)

	if err != nil {
		err = fmt.Errorf("failed to create a db service of type %s:%s, failed to retrieve env vars : %v", dbType, dbVersion, err)

		return nil, err
	}

	var targetURL string
	var ports []model.Port
	if dbType == "postgres" {

		dbEnVars = append(dbEnVars,
			model.EnVar{Key: "POSTGRES_DB", Value: dbName},
			model.EnVar{Key: "POSTGRES_USER", Value: dbUser},
			model.EnVar{Key: "POSTGRES_PASSWORD", Value: dbPass},
		)

		ports = []model.Port{{

			Port:     "5432",
			HostPort: "5432",
			Exposed:  true,
		}}
		targetURL = fmt.Sprintf("%s:%s@localhost:5432/%s", dbUser, dbPass, dbName)

	} else if dbType == "mysql" {
		dbEnVars = append(dbEnVars,
			model.EnVar{Key: "MYSQL_DATABASE", Value: dbName},
		)

		if dbUser == "root" {
			dbEnVars = append(dbEnVars,
				model.EnVar{Key: "MYSQL_ROOT_PASSWORD", Value: dbPass},
			)

		} else {
			dbEnVars = append(dbEnVars,
				model.EnVar{Key: "MYSQL_USER", Value: dbUser},
				model.EnVar{Key: "MYSQL_PASSWORD", Value: dbPass},
			)

		}

		targetURL = fmt.Sprintf("%s:%s@localhost:3306/%s", dbUser, dbPass, dbName)
		ports = []model.Port{{

			Port:     "3306",
			HostPort: "3306",
			Exposed:  true,
		}}
	} else {
		return nil, fmt.Errorf("failed to set DB env vars, unsupported db type %s", dbType)
	}

	return &model.Service{
		Name:        "perun-db",
		Description: "perun loaded db",
		Type:        "docker",
		Build: &model.BuildConfig{
			Type: "db",
			Params: map[string]string{
				"url":        dbURL,
				"target-url": targetURL,
				"type":       dbType,
			},
		},
		Params: map[string]string{
			"image": image,
		},
		Run: &model.RunConfig{
			EnVars: dbEnVars,
			Ports:  ports,
		},
	}, nil
}

func GetDBImage(dbType string, dbVersion string) (string, error) {

	image := ""
	if dbVersion == "" {
		dbVersion = "latest"
	}

	if dbType == "mysql" {
		image = "mysql:" + dbVersion
	} else if dbType == "postgres" {
		image = "postgres:" + dbVersion
	} else {
		return "", fmt.Errorf("failed to retrieve db docker image, type %s not supported", dbType)
	}

	return image, nil
}

func GetSQLEnVars(dbURL string) (string, string, string, string, error) {
	u, err := url.Parse(dbURL)
	if err != nil {
		return "", "", "", "", err
	}

	// Extract database name
	dbName := strings.TrimPrefix(u.Path, "/")

	// Extract username and password
	dbUser := ""
	dbPassword := ""
	if u.User != nil {
		dbUser = u.User.Username()
		dbPassword, _ = u.User.Password()
	}

	return u.Host, dbName, dbUser, dbPassword, nil
}
