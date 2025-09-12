package handlers

import (
	"github.com/stretchr/testify/mock"
)


// MockCollectorService is a mock implementation of CollectorInterface for testing
type MockCollectorService struct {
	mock.Mock
}

func (m *MockCollectorService) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCollectorService) Stop() {
	m.Called()
}

func (m *MockCollectorService) RestartWorker(exchangeID string) error {
	args := m.Called(exchangeID)
	return args.Error(0)
}

func (m *MockCollectorService) IsReady() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockCollectorService) IsInitialized() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockCollectorService) GetStatus() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}
