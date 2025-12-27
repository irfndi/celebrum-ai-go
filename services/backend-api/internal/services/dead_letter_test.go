package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeadLetterEntry_Struct(t *testing.T) {
	now := time.Now()
	nextRetry := now.Add(5 * time.Minute)

	entry := DeadLetterEntry{
		ID:             "test-id-123",
		UserID:         "user-456",
		ChatID:         "789012345",
		MessageType:    "telegram_notification",
		MessageContent: "Test message content",
		ErrorCode:      "RATE_LIMITED",
		ErrorMessage:   "Too many requests",
		Attempts:       2,
		Status:         DeadLetterStatusPending,
		CreatedAt:      now,
		LastAttemptAt:  now,
		NextRetryAt:    &nextRetry,
	}

	assert.Equal(t, "test-id-123", entry.ID)
	assert.Equal(t, "user-456", entry.UserID)
	assert.Equal(t, "789012345", entry.ChatID)
	assert.Equal(t, "telegram_notification", entry.MessageType)
	assert.Equal(t, "Test message content", entry.MessageContent)
	assert.Equal(t, "RATE_LIMITED", entry.ErrorCode)
	assert.Equal(t, "Too many requests", entry.ErrorMessage)
	assert.Equal(t, 2, entry.Attempts)
	assert.Equal(t, DeadLetterStatusPending, entry.Status)
	assert.Equal(t, now, entry.CreatedAt)
	assert.Equal(t, now, entry.LastAttemptAt)
	assert.Equal(t, &nextRetry, entry.NextRetryAt)
}

func TestDeadLetterStats_Struct(t *testing.T) {
	oldestTime := time.Now().Add(-1 * time.Hour)

	stats := DeadLetterStats{
		TotalCount:    100,
		PendingCount:  25,
		RetryingCount: 5,
		FailedCount:   60,
		SuccessCount:  10,
		ByErrorCode: map[string]int{
			"USER_BLOCKED":  30,
			"RATE_LIMITED":  20,
			"NETWORK_ERROR": 10,
		},
		OldestPending: &oldestTime,
	}

	assert.Equal(t, 100, stats.TotalCount)
	assert.Equal(t, 25, stats.PendingCount)
	assert.Equal(t, 5, stats.RetryingCount)
	assert.Equal(t, 60, stats.FailedCount)
	assert.Equal(t, 10, stats.SuccessCount)
	assert.Equal(t, 30, stats.ByErrorCode["USER_BLOCKED"])
	assert.Equal(t, 20, stats.ByErrorCode["RATE_LIMITED"])
	assert.Equal(t, 10, stats.ByErrorCode["NETWORK_ERROR"])
	assert.Equal(t, &oldestTime, stats.OldestPending)
}

func TestDeadLetterStatus_Constants(t *testing.T) {
	assert.Equal(t, DeadLetterStatus("pending"), DeadLetterStatusPending)
	assert.Equal(t, DeadLetterStatus("retrying"), DeadLetterStatusRetrying)
	assert.Equal(t, DeadLetterStatus("failed"), DeadLetterStatusFailed)
	assert.Equal(t, DeadLetterStatus("success"), DeadLetterStatusSuccess)
}

func TestNewDeadLetterService_NilDB(t *testing.T) {
	dls := NewDeadLetterService(nil)
	assert.NotNil(t, dls)
	assert.Nil(t, dls.db)
	assert.NotNil(t, dls.logger)
}

func TestIsRetryableErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "RATE_LIMITED is retryable",
			code:     "RATE_LIMITED",
			expected: true,
		},
		{
			name:     "NETWORK_ERROR is retryable",
			code:     "NETWORK_ERROR",
			expected: true,
		},
		{
			name:     "TIMEOUT is retryable",
			code:     "TIMEOUT",
			expected: true,
		},
		{
			name:     "INTERNAL_ERROR is retryable",
			code:     "INTERNAL_ERROR",
			expected: true,
		},
		{
			name:     "USER_BLOCKED is not retryable",
			code:     "USER_BLOCKED",
			expected: false,
		},
		{
			name:     "CHAT_NOT_FOUND is not retryable",
			code:     "CHAT_NOT_FOUND",
			expected: false,
		},
		{
			name:     "INVALID_REQUEST is not retryable",
			code:     "INVALID_REQUEST",
			expected: false,
		},
		{
			name:     "UNKNOWN is not retryable",
			code:     "UNKNOWN",
			expected: false,
		},
		{
			name:     "Empty string is not retryable",
			code:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableErrorCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPow(t *testing.T) {
	tests := []struct {
		base     int
		exp      int
		expected int
	}{
		{base: 2, exp: 0, expected: 1},
		{base: 2, exp: 1, expected: 2},
		{base: 2, exp: 2, expected: 4},
		{base: 2, exp: 3, expected: 8},
		{base: 3, exp: 0, expected: 1},
		{base: 3, exp: 1, expected: 3},
		{base: 3, exp: 2, expected: 9},
		{base: 3, exp: 3, expected: 27},
		{base: 5, exp: 3, expected: 125},
	}

	for _, tt := range tests {
		result := pow(tt.base, tt.exp)
		assert.Equal(t, tt.expected, result)
	}
}

func TestDeadLetterEntry_NilNextRetryAt(t *testing.T) {
	entry := DeadLetterEntry{
		ID:          "test-id",
		UserID:      "user-id",
		ChatID:      "chat-id",
		MessageType: "test",
		Status:      DeadLetterStatusFailed,
		NextRetryAt: nil,
	}

	assert.Nil(t, entry.NextRetryAt)
}

func TestDeadLetterStats_EmptyByErrorCode(t *testing.T) {
	stats := DeadLetterStats{
		TotalCount:    0,
		PendingCount:  0,
		RetryingCount: 0,
		FailedCount:   0,
		SuccessCount:  0,
		ByErrorCode:   make(map[string]int),
		OldestPending: nil,
	}

	assert.Equal(t, 0, stats.TotalCount)
	assert.Empty(t, stats.ByErrorCode)
	assert.Nil(t, stats.OldestPending)
}
