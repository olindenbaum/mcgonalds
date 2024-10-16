package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockServerManager struct {
	mock.Mock
}

var _ ServerManager = (*MockServerManager)(nil)

func (m *MockServerManager) CreateServer(name, path string) error {
	args := m.Called(name, path)
	return args.Error(0)
}

func (m *MockServerManager) GetServer(name string) (*server.Server, error) {
	args := m.Called(name)
	return args.Get(0).(*server.Server), args.Error(1)
}

// Implement other ServerManager methods...

func TestCreateServer(t *testing.T) {
	mockSM := new(MockServerManager)
	h := NewHandler(&gorm.DB{}, mockSM, &config.Config{})

	mockSM.On("CreateServer", "test_server", "/path/to/server").Return(nil)

	body := strings.NewReader(`{"name": "test_server", "path": "/path/to/server"}`)
	req, _ := http.NewRequest("POST", "/servers", body)
	rr := httptest.NewRecorder()

	h.CreateServer(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	mockSM.AssertExpectations(t)
}

func TestGetServer(t *testing.T) {
	mockSM := new(MockServerManager)
	h := NewHandler(&gorm.DB{}, mockSM, &config.Config{})

	mockServer := &server.Server{}
	mockSM.On("GetServer", "test_server").Return(mockServer, nil)

	req, _ := http.NewRequest("GET", "/servers/test_server", nil)
	rr := httptest.NewRecorder()

	vars := map[string]string{
		"name": "test_server",
	}
	req = mux.SetURLVars(req, vars)

	h.GetServer(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var responseServer server.Server
	json.Unmarshal(rr.Body.Bytes(), &responseServer)
	assert.Equal(t, mockServer, &responseServer)
	mockSM.AssertExpectations(t)
}

// Add more tests for other handler methods
