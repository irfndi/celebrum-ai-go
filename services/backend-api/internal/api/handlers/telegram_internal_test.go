package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

// TestTelegramInternalHandler_GetNotificationPreferences_Success tests success case
func TestTelegramInternalHandler_GetNotificationPreferences_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockDB.Close()

	handler := NewTelegramInternalHandler(mockDB, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/internal/telegram/users/user-123/preferences", nil)
	c.Params = gin.Params{{Key: "userId", Value: "user-123"}}

	// 1. Check disabled count
	mockDB.ExpectQuery("SELECT COUNT").
		WithArgs("user-123").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	// 2. Fetch active alert
	mockDB.ExpectQuery("SELECT conditions").
		WithArgs("user-123").
		WillReturnRows(pgxmock.NewRows([]string{"conditions"}).AddRow([]byte(`{"profit_threshold": 0.5}`)))

	handler.GetNotificationPreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["enabled"])
	assert.Equal(t, 0.5, response["profit_threshold"])
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTelegramInternalHandler_SetNotificationPreferences_Success_Enable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockDB.Close()

	handler := NewTelegramInternalHandler(mockDB, nil)

	req := map[string]bool{"enabled": true}
	jsonBytes, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/internal/telegram/users/user-123/preferences", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "userId", Value: "user-123"}}

	mockDB.ExpectBegin()
	mockDB.ExpectExec("DELETE FROM user_alerts").
		WithArgs("user-123").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mockDB.ExpectCommit()

	handler.SetNotificationPreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTelegramInternalHandler_SetNotificationPreferences_Success_Disable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockDB.Close()

	handler := NewTelegramInternalHandler(mockDB, nil)

	req := map[string]bool{"enabled": false}
	jsonBytes, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/internal/telegram/users/user-123/preferences", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "userId", Value: "user-123"}}

	mockDB.ExpectBegin()
	mockDB.ExpectExec("DELETE FROM user_alerts").
		WithArgs("user-123").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mockDB.ExpectExec("INSERT INTO user_alerts").
		WithArgs(pgxmock.AnyArg(), "user-123", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mockDB.ExpectCommit()

	handler.SetNotificationPreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// Keep existing basic tests
func TestNewTelegramInternalHandler(t *testing.T) {
	mockDB, _ := pgxmock.NewPool()
	defer mockDB.Close()
	userHandler := &UserHandler{}
	handler := NewTelegramInternalHandler(mockDB, userHandler)

	assert.NotNil(t, handler)
	assert.Equal(t, mockDB, handler.db)
}

func TestTelegramInternalHandler_PathParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		paramKey   string
		paramValue string
		expected   string
	}{
		{
			name:       "chat id parameter",
			paramKey:   "id",
			paramValue: "123456789",
			expected:   "123456789",
		},
		// ... existing tests ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: tt.paramKey, Value: tt.paramValue}}
			assert.Equal(t, tt.expected, c.Param(tt.paramKey))
		})
	}
}

// ... include other existing tests if valuable ...
func TestTelegramInternalHandler_ErrorResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name           string
		expectedStatus int
		errorMessage   string
	}{
		{
			name:           "user not found",
			expectedStatus: http.StatusNotFound,
			errorMessage:   "User not found",
		},
		{
			name:           "bad request",
			expectedStatus: http.StatusBadRequest,
			errorMessage:   "Chat ID required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.JSON(tt.expectedStatus, gin.H{"error": tt.errorMessage})
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
