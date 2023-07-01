package services

import (
	"testing"

	"github.com/perun-cloud-inc/perunctl/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DummyPersistenceService struct {
	mock.Mock
}

func (m *DummyPersistenceService) PersistWorkspace(ws *model.Workspace) error {
	args := m.Called(ws)
	return args.Error(0)
}
func (m *DummyPersistenceService) ClearWorkspace(ws *model.Workspace) error {
	args := m.Called(ws)
	return args.Error(0)
}

func (m *DummyPersistenceService) GetWorkspace(name string) (*model.Workspace, error) {
	args := m.Called(name)
	return args.Get(0).(*model.Workspace), args.Error(1)
}
func (m *DummyPersistenceService) ListWorkspaces() ([]*model.Workspace, error) {
	args := m.Called()
	return args.Get(0).([]*model.Workspace), args.Error(1)

}
func (m *DummyPersistenceService) LoadEnvironment(envPath string) (*model.Environment, error) {
	args := m.Called(envPath)
	return args.Get(0).(*model.Environment), args.Error(1)
}

type DummyValidationService struct {
	mock.Mock
}

func (m *DummyValidationService) ValidateWorkspace(ws *model.Workspace) error {
	args := m.Called(ws)
	return args.Error(0)
}

type DummyEnvironmentService struct {
	mock.Mock
}

func (m *DummyEnvironmentService) CreateEnvironment(name string) (*model.Environment, error) {
	args := m.Called(name)
	return args.Get(0).(*model.Environment), args.Error(1)
}

func (m *DummyEnvironmentService) ActivateEnvironment(env *model.Environment) error {
	args := m.Called(env)
	return args.Error(0)
}

func (m *DummyEnvironmentService) DeactivateEnvironment(env *model.Environment) error {
	args := m.Called(env)
	return args.Error(0)
}

func (m *DummyEnvironmentService) DestroyEnvironment(env *model.Environment) error {
	args := m.Called(env)
	return args.Error(0)
}

func (m *DummyEnvironmentService) SyncEnvironment(env *model.Environment) error {
	args := m.Called(env)
	return args.Error(0)
}

type DummyAnalyzerService struct {
	mock.Mock
}

func (m *DummyAnalyzerService) AnalyzeEnvironment(env *model.Environment) (*model.Environment, error) {
	args := m.Called(env)
	return args.Get(0).(*model.Environment), args.Error(1)
}

func (m *DummyAnalyzerService) AnalyzeService(service *model.Service) (*model.Service, error) {
	args := m.Called(service)
	return args.Get(0).(*model.Service), args.Error(1)
}

// TestCreateWorkspace
func TestCreateWorkspace(t *testing.T) {
	wss := GetWorkspaceService()

	ps := new(DummyPersistenceService)
	vs := new(DummyValidationService)

	wss.PersistenceService = ps
	ps.On("PersistWorkspace", mock.Anything).Return(nil)

	wss.ValidationService = vs
	vs.On("ValidateWorkspace", mock.Anything).Return(nil)

	ws, err := wss.CreateWorkspace("test")

	assert.Equal(t, "test", ws.Name)
	assert.Nil(t, err)

	ps.AssertExpectations(t)
	vs.AssertExpectations(t)

}

func TestWSGetWorkspace(t *testing.T) {
	wss := GetWorkspaceService()

	ps := new(DummyPersistenceService)

	wss.PersistenceService = ps
	expectedWS := &model.Workspace{
		Name: "test",
	}

	ps.On("GetWorkspace", "test").Return(expectedWS, nil)

	ws, err := wss.GetWorkspace("test")

	assert.Equal(t, expectedWS, ws)
	assert.Nil(t, err)

	ps.AssertExpectations(t)

}

func TestWSListWorkspaces(t *testing.T) {

	wss := GetWorkspaceService()

	ps := new(DummyPersistenceService)

	wss.PersistenceService = ps
	expectedWSList := []*model.Workspace{{
		Name: "test",
	}}

	ps.On("ListWorkspaces").Return(expectedWSList, nil)

	wsList, err := wss.ListWorkspaces()

	assert.Equal(t, expectedWSList, wsList)
	assert.Nil(t, err)

	ps.AssertExpectations(t)

}

func TestDestroyWorkspace(t *testing.T) {
	wss := GetWorkspaceService()

	ps := new(DummyPersistenceService)

	wss.PersistenceService = ps
	testEnv := &model.Environment{
		Name: "testenv",
	}
	expectedWS := &model.Workspace{
		Name: "test",
		Environments: []*model.Environment{
			testEnv,
		},
	}

	ps.On("GetWorkspace", "test").Return(expectedWS, nil)
	ps.On("ClearWorkspace", expectedWS).Return(nil)

	es := new(DummyEnvironmentService)
	wss.EnvironmentService = es

	es.On("DestroyEnvironment", testEnv).Return(nil)
	err := wss.DestroyWorkspace("test")

	assert.Nil(t, err)

	ps.AssertExpectations(t)
	es.AssertExpectations(t)

}

func TestActivateEnvironment(t *testing.T) {

	wss := GetWorkspaceService()

	ps := new(DummyPersistenceService)

	wss.PersistenceService = ps
	testEnv := &model.Environment{
		Name: "testEnv",
	}
	expectedWS := &model.Workspace{
		Name: "test",
		Environments: []*model.Environment{
			testEnv,
		},
	}

	ps.On("GetWorkspace", "test").Return(expectedWS, nil)

	ps.On("PersistWorkspace", expectedWS).Return(nil)
	es := new(DummyEnvironmentService)
	wss.EnvironmentService = es

	es.On("ActivateEnvironment", testEnv).Return(nil)

	err := wss.ActivateEnvironment("test", "testEnv")
	assert.Nil(t, err)

	ps.AssertExpectations(t)
	es.AssertExpectations(t)

}

func TestDeactivateEnvironment(t *testing.T) {

	wss := GetWorkspaceService()

	ps := new(DummyPersistenceService)

	wss.PersistenceService = ps
	testEnv := &model.Environment{
		Name: "testEnv",
	}
	expectedWS := &model.Workspace{
		Name: "test",
		Environments: []*model.Environment{
			testEnv,
		},
	}

	ps.On("GetWorkspace", "test").Return(expectedWS, nil)

	ps.On("PersistWorkspace", expectedWS).Return(nil)
	es := new(DummyEnvironmentService)
	wss.EnvironmentService = es

	es.On("DeactivateEnvironment", testEnv).Return(nil)

	err := wss.DeactivateEnvironment("test", "testEnv")
	assert.Nil(t, err)

	ps.AssertExpectations(t)
	es.AssertExpectations(t)
}

func TestSynchronizeEnvironment(t *testing.T) {
	wss := GetWorkspaceService()

	ps := new(DummyPersistenceService)

	wss.PersistenceService = ps
	testEnv := &model.Environment{
		Name: "testEnv",
	}
	expectedWS := &model.Workspace{
		Name: "test",
		Environments: []*model.Environment{
			testEnv,
		},
	}

	ps.On("GetWorkspace", "test").Return(expectedWS, nil)

	es := new(DummyEnvironmentService)
	wss.EnvironmentService = es

	es.On("SyncEnvironment", testEnv).Return(nil)

	err := wss.SynchronizeEnvironment("test", "testEnv")
	assert.Nil(t, err)

	ps.AssertExpectations(t)
	es.AssertExpectations(t)
}

func TestDestroyEnvironment(t *testing.T) {
	wss := GetWorkspaceService()

	err := wss.DestroyEnvironment("test", "testEnv")

	assert.Nil(t, err)
}
