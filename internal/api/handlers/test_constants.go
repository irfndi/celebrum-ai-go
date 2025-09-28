package handlers

import (
	"crypto/rand"
	"math/big"
)

// generateTestPassword creates a random password for testing to avoid hardcoded secrets
// This addresses GitGuardian security alerts about hardcoded test credentials
func generateTestPassword() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, 12)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[n.Int64()]
	}
	return string(b)
}

// Test constants for secure password handling across all handler tests
// These now use dynamically generated passwords to avoid hardcoded secrets
var (
	TestValidPassword     = generateTestPassword()
	TestWrongPassword     = generateTestPassword()
	TestDifferentPassword = generateTestPassword()
	TestPassword123       = generateTestPassword()
	TestCorrectPassword   = generateTestPassword()
)
