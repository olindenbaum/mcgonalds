package server_manager

import (
	"testing"

	"github.com/olindenbaum/mcgonalds/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) Create(value interface{}) *gorm.DB {
	args := m.Called(value)
	return args.Get(0).(*gorm.DB)
}

func (m *MockDB) Where(query interface{}, args ...interface{}) *gorm.DB {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(*gorm.DB)
}

func (m *MockDB) First(dest interface{}) *gorm.DB {
	args := m.Called(dest)
	return args.Get(0).(*gorm.DB)
}

func (m *MockDB) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	args := m.Called(value, conds)
	return args.Get(0).(*gorm.DB)
}

func TestCreateServer(t *testing.T) {
	mockDB := new(MockDB)
	sm := NewServerManager(mockDB)

	mockDB.On("Create", mock.AnythingOfType("*model.Server")).Return(&gorm.DB{Error: nil})

	err := sm.CreateServer("test_server", "/path/to/server")

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestGetServer(t *testing.T) {
	mockDB := new(MockDB)
	sm := NewServerManager(mockDB)

	mockDB.On("Where", "name = ?", "test_server").Return(mockDB)
	mockDB.On("First", mock.AnythingOfType("*model.Server")).Return(&gorm.DB{Error: nil}).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*model.Server)
		arg.Name = "test_server"
		arg.Path = "/path/to/server"
	})

	server, err := sm.GetServer("test_server")

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "test_server", server.model.Name)
	mockDB.AssertExpectations(t)
}

// Add more tests for other ServerManager methods
